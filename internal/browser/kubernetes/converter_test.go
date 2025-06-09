package kubernetes

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/selebrow/selebrow/mocks"
	"github.com/selebrow/selebrow/pkg/models"
)

const tpl1 = `apiVersion: v1
kind: Pod
metadata:
  generateName: browser-{{ .CIEnvironment.ProjectNamespace }}-{{ .CIEnvironment.JobID }}-
  labels:
{{- range $k,$v := .Options.Labels }}
    {{ $k }}: {{ quote $v }}
{{- end }}
  annotations:
    project-name: {{ .CIEnvironment.ProjectName }}
spec:
  containers:
  - name: browser
    image: {{ .Browser.Image }}
{{- with .Browser.Cmd }}
    command:
{{ toYaml . | indent 6 }}
{{- end }}
    env:
{{- range $k,$v := mergeOverwrite .Browser.Env .Values.env }}
    - name: {{ $k }}
      value: {{ quote $v }}
{{- end }}
{{- range $k,$v := .Options.Env }}
    - name: {{ $k }}
      value: {{ quote $v }}
{{- end }}
    ports:
{{- range $k,$v := .Browser.Ports }}
    - containerPort: {{ $v }}
      name: {{ quote $k }}
      protocol: TCP
{{- end }}
    readinessProbe:
      httpGet:
        path: {{ .Browser.Path }}/ready
        port: {{ .Browser.Ports.browser }}    
{{- with .Browser.Limits }}
    resources:
      limits:
{{ toYaml . | indent 8 }}
{{- end }}
  hostAliases:
{{- range $k,$v := .Options.Hosts }} 
  - ip: {{ $k | quote }}
    hostnames:
{{ toYaml $v | indent 6 }}
{{- end }}`

const values1 = `
env:
    env1: val1
    env2: overridedval2
`

func TestTemplatedBrowserConverter_ToPod(t *testing.T) {
	g := NewWithT(t)

	cfg := setupConfig()

	caps := new(mocks.Capabilities)
	caps.EXPECT().GetVersion().Return("").Once()
	caps.EXPECT().GetEnvs().Return([]string{"env3=capsval3"}).Once()
	caps.EXPECT().IsVNCEnabled().Return(true)
	caps.EXPECT().GetResolution().Return("320x200")
	caps.EXPECT().GetLabels().Return(map[string]string{"k1": "v1"})
	caps.EXPECT().GetHosts().Return([]string{"aaaa:1.2.3.4", "bbbb:1.2.3.4", "bad"})

	vers := models.BrowserImageConfig{
		Image:          "repo/nutscrape",
		Cmd:            []string{"run", "-opt"},
		DefaultVersion: "3",
		VersionTags: map[string]string{
			"3": "ver3",
		},
		Ports: map[models.ContainerPort]int{
			models.BrowserPort: 1234,
			models.VNCPort:     1111,
		},
		Path: "/wd",
		Env: map[string]string{
			"env2": "val2",
		},
		Limits: map[string]string{
			"cpu":    "1",
			"memory": "2Gi",
		},
	}

	cnv, err := NewTemplatedBrowserConverter(cfg, tpl1, []byte(values1))
	g.Expect(err).ToNot(HaveOccurred())

	pod, err := cnv.ToPod(vers, caps)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(pod).To(Equal(v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "browser-test-12345-",
			Labels: map[string]string{
				"lineage": "321",
				"k1":      "v1",
			},
			Annotations: map[string]string{
				"project-name": "test-proj",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "browser",
					Image:   "repo/nutscrape:ver3",
					Command: []string{"run", "-opt"},
					Ports: []v1.ContainerPort{
						{
							Name:          "browser",
							ContainerPort: 1234,
							Protocol:      "TCP",
						},
						{
							Name:          string(models.VNCPort),
							ContainerPort: 1111,
							Protocol:      "TCP",
						},
					},
					Env: []v1.EnvVar{
						{
							Name:  "env1",
							Value: "val1",
						},
						{
							Name:  "env2",
							Value: "overridedval2",
						},
						{
							Name:  "ENABLE_VNC",
							Value: "true",
						},
						{
							Name:  "SCREEN_RESOLUTION",
							Value: "320x200",
						},
						{
							Name:  "env3",
							Value: "capsval3",
						},
					},
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"cpu":    resource.MustParse("1"),
							"memory": resource.MustParse("2Gi"),
						},
					},
					ReadinessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/wd/ready",
								Port: intstr.FromInt(1234),
							},
						},
					},
				},
			},
			HostAliases: []v1.HostAlias{
				{
					IP:        "1.2.3.4",
					Hostnames: []string{"aaaa", "bbbb"},
				},
			},
		},
	}))

	cfg.AssertExpectations(t)
	caps.AssertExpectations(t)
}

func TestTemplatedBrowserConverter_ToPod_BadVersion(t *testing.T) {
	g := NewWithT(t)

	cfg := setupConfig()

	caps := new(mocks.Capabilities)
	caps.EXPECT().GetVersion().Return("455").Once()
	caps.EXPECT().IsVNCEnabled().Return(false).Once()

	cnv, err := NewTemplatedBrowserConverter(cfg, tpl1, []byte(values1))
	g.Expect(err).ToNot(HaveOccurred())

	vers := models.BrowserImageConfig{
		DefaultVersion: "3",
		VersionTags: map[string]string{
			"3": "ver3",
		},
	}
	_, err = cnv.ToPod(vers, caps)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(models.ErrorWithCode).Code()).To(Equal(http.StatusBadRequest))

	cfg.AssertExpectations(t)
	caps.AssertExpectations(t)
}

func setupConfig() *mocks.BrowserConverterConfig {
	cfg := new(mocks.BrowserConverterConfig)
	cfg.EXPECT().ProjectNamespace().Return("test").Once()
	cfg.EXPECT().ProjectName().Return("test-proj").Once()
	cfg.EXPECT().JobID().Return("12345").Once()
	cfg.EXPECT().Lineage().Return("321").Once()
	return cfg
}
