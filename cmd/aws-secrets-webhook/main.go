package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/doitintl/kube-secrets-init/cmd/aws-secrets-webhook/registry"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	whcontext "github.com/slok/kubewebhook/pkg/webhook/context"
	"github.com/slok/kubewebhook/pkg/webhook/mutating"
	"github.com/urfave/cli"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubernetesConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	// secretsInitContainer is the default secrets-init container from which to pull the
	// secrets-init binary.
	secretsInitImage = "doitintl/secrets-init:latest"

	// binVolumeName is the name of the volume where the secrets-init binary is stored.
	binVolumeName = "secrets-init-bin"

	// binVolumePath is the mount path where the secrets-init binary can be found.
	binVolumePath = "/secrets-init/bin/"
)

var (
	// Version contains the current version.
	Version = "dev"
	// BuildDate contains a string with the build date.
	BuildDate = "unknown"
)

type mutatingWebhook struct {
	k8sClient  kubernetes.Interface
	registry   registry.ImageRegistry
	image      string
	pullPolicy string
	volumeName string
	volumePath string
}

var logger *log.Logger

func newK8SClient() (kubernetes.Interface, error) {
	kubeConfig, err := kubernetesConfig.GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(kubeConfig)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func serveMetrics(addr string) {
	logger.Infof("Telemetry on http://%s", addr)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		logger.Fatalf("error serving telemetry: %s", err)
	}
}

func handlerFor(config mutating.WebhookConfig, mutator mutating.MutatorFunc, recorder metrics.Recorder, logger *log.Logger) http.Handler {
	webhook, err := mutating.NewWebhook(config, mutator, nil, recorder, logger)
	if err != nil {
		logger.Fatalf("error creating webhook: %s", err)
	}

	handler, err := whhttp.HandlerFor(webhook)
	if err != nil {
		logger.Fatalf("error creating webhook: %s", err)
	}

	return handler
}

func hasSecretsPrefix(value string) bool {
	return strings.HasPrefix(value, "arn:aws:secretsmanager") || (strings.HasPrefix(value, "arn:aws:ssm") && strings.Contains(value, ":parameter/"))
}

func (mw *mutatingWebhook) getDataFromConfigmap(cmName string, ns string) (map[string]string, error) {
	configMap, err := mw.k8sClient.CoreV1().ConfigMaps(ns).Get(cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap.Data, nil
}

func (mw *mutatingWebhook) getDataFromSecret(secretName string, ns string) (map[string][]byte, error) {
	secret, err := mw.k8sClient.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (mw *mutatingWebhook) lookForEnvFrom(envFrom []corev1.EnvFromSource, ns string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	for _, ef := range envFrom {
		if ef.ConfigMapRef != nil {
			data, err := mw.getDataFromConfigmap(ef.ConfigMapRef.Name, ns)
			if err != nil {
				if apierrors.IsNotFound(err) && ef.ConfigMapRef.Optional != nil && *ef.ConfigMapRef.Optional {
					continue
				} else {
					return envVars, err
				}
			}
			for key, value := range data {
				if hasSecretsPrefix(value) {
					envFromCM := corev1.EnvVar{
						Name:  key,
						Value: value,
					}
					envVars = append(envVars, envFromCM)
				}
			}
		}
		if ef.SecretRef != nil {
			data, err := mw.getDataFromSecret(ef.SecretRef.Name, ns)
			if err != nil {
				if apierrors.IsNotFound(err) && ef.SecretRef.Optional != nil && *ef.SecretRef.Optional {
					continue
				} else {
					return envVars, err
				}
			}
			for key, value := range data {
				if hasSecretsPrefix(string(value)) {
					envFromSec := corev1.EnvVar{
						Name:  key,
						Value: string(value),
					}
					envVars = append(envVars, envFromSec)
				}
			}
		}
	}
	return envVars, nil
}

func (mw *mutatingWebhook) lookForValueFrom(env corev1.EnvVar, ns string) (*corev1.EnvVar, error) {
	if env.ValueFrom.ConfigMapKeyRef != nil {
		data, err := mw.getDataFromConfigmap(env.ValueFrom.ConfigMapKeyRef.Name, ns)
		if err != nil {
			return nil, err
		}
		if hasSecretsPrefix(data[env.ValueFrom.ConfigMapKeyRef.Key]) {
			fromCM := corev1.EnvVar{
				Name:  env.Name,
				Value: data[env.ValueFrom.ConfigMapKeyRef.Key],
			}
			return &fromCM, nil
		}
	}
	if env.ValueFrom.SecretKeyRef != nil {
		data, err := mw.getDataFromSecret(env.ValueFrom.SecretKeyRef.Name, ns)
		if err != nil {
			return nil, err
		}
		if hasSecretsPrefix(string(data[env.ValueFrom.SecretKeyRef.Key])) {
			fromSecret := corev1.EnvVar{
				Name:  env.Name,
				Value: string(data[env.ValueFrom.SecretKeyRef.Key]),
			}
			return &fromSecret, nil
		}
	}
	return nil, nil
}

func (mw *mutatingWebhook) mutateContainers(containers []corev1.Container, podSpec *corev1.PodSpec, ns string) (bool, error) {
	mutated := false

	for i, container := range containers {
		var envVars []corev1.EnvVar
		if len(container.EnvFrom) > 0 {
			envFrom, err := mw.lookForEnvFrom(container.EnvFrom, ns)
			if err != nil {
				return false, err
			}
			envVars = append(envVars, envFrom...)
		}

		for _, env := range container.Env {
			if hasSecretsPrefix(env.Value) {
				envVars = append(envVars, env)
			}
			if env.ValueFrom != nil {
				valueFrom, err := mw.lookForValueFrom(env, ns)
				if err != nil {
					return false, err
				}
				if valueFrom == nil {
					continue
				}
				envVars = append(envVars, *valueFrom)
			}
		}

		if len(envVars) == 0 {
			continue
		}

		mutated = true

		args := container.Command

		// the container has no explicitly specified command
		if len(args) == 0 {
			imageConfig, err := mw.registry.GetImageConfig(mw.k8sClient, ns, &container, podSpec)
			if err != nil {
				return false, err
			}

			args = append(args, imageConfig.Entrypoint...)

			// If no Args are defined we can use the Docker CMD from the image
			// https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#notes
			if len(container.Args) == 0 {
				args = append(args, imageConfig.Cmd...)
			}
		}

		args = append(args, container.Args...)

		container.Command = []string{"/secrets-init/secrets-init"}
		container.Args = args

		container.VolumeMounts = append(container.VolumeMounts, []corev1.VolumeMount{
			{
				Name:      "secrets-init",
				MountPath: "/secrets-init/",
			},
		}...)

		containers[i] = container
	}

	return mutated, nil
}

func (mw *mutatingWebhook) mutatePod(pod *corev1.Pod, ns string, image string, pullPolicy string, volumeName string, volumePath string, dryRun bool) error {

	logger.Debugf("Successfully connected to the API")

	initContainersMutated, err := mw.mutateContainers(pod.Spec.InitContainers, &pod.Spec, ns)
	if err != nil {
		return err
	}

	if initContainersMutated {
		logger.Debugf("Successfully mutated pod init containers")
	} else {
		logger.Debugf("No pod init containers were mutated")
	}

	containersMutated, err := mw.mutateContainers(pod.Spec.Containers, &pod.Spec, ns)
	if err != nil {
		return err
	}

	if containersMutated {
		logger.Debugf("Successfully mutated pod containers")
	} else {
		logger.Debugf("No pod containers were mutated")
	}

	containerEnvVars := []corev1.EnvVar{}
	containerVolMounts := []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: volumePath,
		},
	}

	if initContainersMutated || containersMutated {
		pod.Spec.InitContainers = append(getInitContainers(pod.Spec.Containers, pod.Spec.SecurityContext, initContainersMutated, containersMutated, containerEnvVars, containerVolMounts, image, pullPolicy, volumeName, volumePath), pod.Spec.InitContainers...)
		logger.Debugf("Successfully appended pod init containers to spec")

		pod.Spec.Volumes = append(pod.Spec.Volumes, getVolumes(pod.Spec.Volumes, volumeName, logger)...)
		logger.Debugf("Successfully appended pod spec volumes")
	}

	return nil
}

func getVolumes(existingVolumes []corev1.Volume, volumeName string, logger *log.Logger) []corev1.Volume {
	logger.Debugf("Add generic volumes to podspec")
	volumes := []corev1.Volume{
		{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
	}
	return volumes
}

func getInitContainers(originalContainers []corev1.Container, podSecurityContext *corev1.PodSecurityContext, initContainersMutated bool, containersMutated bool, containerEnvVars []corev1.EnvVar, containerVolMounts []corev1.VolumeMount, image string, pullPolicy string, volumeName string, volumePath string) []corev1.Container {
	var containers = []corev1.Container{}

	if initContainersMutated || containersMutated {
		containers = append(containers, corev1.Container{
			Name:            "copy-secrets-init",
			Image:           image,
			ImagePullPolicy: corev1.PullPolicy(pullPolicy),
			Command:         []string{"sh", "-c", fmt.Sprintf("cp /usr/local/bin/secrets-init %s", volumePath)},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: volumePath,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		})
	}

	return containers
}

func init() {
	logger = log.New()
	// set log level
	logger.SetLevel(log.WarnLevel)
	logger.SetFormatter(&log.TextFormatter{})
}

func before(c *cli.Context) error {
	// set debug log level
	switch level := c.GlobalString("log-level"); level {
	case "debug", "DEBUG":
		logger.SetLevel(log.DebugLevel)
	case "info", "INFO":
		logger.SetLevel(log.InfoLevel)
	case "warning", "WARNING":
		logger.SetLevel(log.WarnLevel)
	case "error", "ERROR":
		logger.SetLevel(log.ErrorLevel)
	case "fatal", "FATAL":
		logger.SetLevel(log.FatalLevel)
	case "panic", "PANIC":
		logger.SetLevel(log.PanicLevel)
	default:
		logger.SetLevel(log.WarnLevel)
	}
	// set log formatter to JSON
	if c.GlobalBool("json") {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	return nil
}

func (mw *mutatingWebhook) secretsMutator(ctx context.Context, obj metav1.Object) (bool, error) {
	switch v := obj.(type) {
	case *corev1.Pod:
		return false, mw.mutatePod(v, whcontext.GetAdmissionRequest(ctx).Namespace, mw.image, mw.pullPolicy, mw.volumeName, mw.volumePath, whcontext.IsAdmissionRequestDryRun(ctx))
	// case *corev1.Secret:
	// 	if _, ok := obj.GetAnnotations()["vault.security.banzaicloud.io/vault-addr"]; ok {
	// 		return false, mutateSecret(v, parseVaultConfig(obj), whcontext.GetAdmissionRequest(ctx).Namespace)
	// 	}
	// 	return false, nil
	// case *corev1.ConfigMap:
	// 	if _, ok := obj.GetAnnotations()["vault.security.banzaicloud.io/mutate-configmap"]; ok {
	// 		return false, mutateConfigMap(v, parseVaultConfig(obj), whcontext.GetAdmissionRequest(ctx).Namespace)
	// 	}
	// 	return false, nil
	default:
		return false, nil
	}
}

// mutation webhook server
func runWebhook(c *cli.Context) error {
	k8sClient, err := newK8SClient()
	if err != nil {
		logger.Fatalf("error creating k8s client: %s", err)
	}

	mutatingWebhook := mutatingWebhook{
		k8sClient:  k8sClient,
		registry:   registry.NewRegistry(c.Bool("registry-skip-verify"), c.String("docker-config-json-key"), c.String("default-image-pull-secret"), c.String("default-image-pull-secret-namespace")),
		image:      c.String("image"),
		pullPolicy: c.String("pull-policy"),
		volumeName: c.String("volume-name"),
		volumePath: c.String("volume-path"),
	}

	mutator := mutating.MutatorFunc(mutatingWebhook.secretsMutator)
	metricsRecorder := metrics.NewPrometheus(prometheus.DefaultRegisterer)

	podHandler := handlerFor(mutating.WebhookConfig{Name: "init-secrets-pods", Obj: &corev1.Pod{}}, mutator, metricsRecorder, logger)

	mux := http.NewServeMux()
	mux.Handle("/pods", podHandler)
	mux.Handle("/healthz", http.HandlerFunc(healthzHandler))

	telemetryAddress := c.String("telemetry-listen-address")
	listenAddress := c.String("listen-address")
	tlsCertFile := c.String("tls-cert-file")
	tlsPrivateKeyFile := c.String("tls-private-key-file")

	if len(telemetryAddress) > 0 {
		// Serving metrics without TLS on separated address
		go serveMetrics(telemetryAddress)
	} else {
		mux.Handle("/metrics", promhttp.Handler())
	}

	if tlsCertFile == "" && tlsPrivateKeyFile == "" {
		logger.Infof("Listening on http://%s", listenAddress)
		err = http.ListenAndServe(listenAddress, mux)
	} else {
		logger.Infof("Listening on https://%s", listenAddress)
		err = http.ListenAndServeTLS(listenAddress, tlsCertFile, tlsPrivateKeyFile, mux)
	}

	if err != nil {
		logger.Fatalf("error serving webhook: %s", err)
	}

	return nil
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("version: %s\n", c.App.Version)
		fmt.Printf("  build date: %s\n", BuildDate)
		fmt.Printf("  built with: %s\n", runtime.Version())
	}
	app := cli.NewApp()
	app.Name = "kube-secrets-init"
	app.Version = Version
	app.Authors = []cli.Author{
		{
			Name:  "Alexei Ledenev",
			Email: "alexei.led@gmail.com",
		},
	}
	app.Usage = "kube-secrets-init is a Kubernetes mutation controller that injects a sidecar init container with a secrets-init on-board"
	app.Before = before
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level",
			Usage:  "set log level (debug, info, warning(*), error, fatal, panic)",
			Value:  "warning",
			EnvVar: "LOG_LEVEL",
		},
		cli.BoolFlag{
			Name:   "json",
			Usage:  "produce log in JSON format: Logstash and Splunk friendly",
			EnvVar: "LOG_JSON",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name: "server",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "listen-address",
					Usage: "webhook server listen address",
					Value: ":8443",
				},
				cli.StringFlag{
					Name:  "telemetry-listen-address",
					Usage: "specify a dedicated prometheus metrics listen address (using listen-address, if empty)",
				},
				cli.StringFlag{
					Name:  "tls-cert-file",
					Usage: "TLS certificate file",
				},
				cli.StringFlag{
					Name:  "tls-private-key-file",
					Usage: "TLS private key file",
				},
				cli.StringFlag{
					Name:  "image",
					Usage: "Docker image with secrets-init utility on board",
					Value: secretsInitImage,
				},
				cli.StringFlag{
					Name:  "pull-policy",
					Usage: "Docker image pull policy",
					Value: string(corev1.PullIfNotPresent),
				},
				cli.BoolFlag{
					Name:  "registry-skip-verify",
					Usage: "use insecure Docker registry",
				},
				cli.StringFlag{
					Name:  "docker-config-json-key",
					Usage: "key of the required data for SecretTypeDockerConfigJson secret",
					Value: corev1.DockerConfigJsonKey,
				},
				cli.StringFlag{
					Name:  "default_image_pull_secret",
					Usage: "default image pull secret",
				},
				cli.StringFlag{
					Name:  "default_image_pull_secret_namespace",
					Usage: "default image pull secret namespace",
				},
				cli.StringFlag{
					Name:  "volume-name",
					Usage: "mount volume name",
					Value: binVolumeName,
				},
				cli.StringFlag{
					Name:  "volume-path",
					Usage: "mount volume path",
					Value: binVolumePath,
				},
			},
			Usage:       "mutation admission webhook",
			Description: "run mutation admission webhook server",
			Action:      runWebhook,
		},
	}

	// run main command
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
