package main

type Protocol string

const (
	TCP Protocol = "tcp"
	UDP          = "udp"
)

type Port struct {
	Target    uint     `yaml:"target"`
	Published uint     `yaml:"published"`
	Protocol  Protocol `yaml:"protocol"`
	Mode      string   `yaml:"mode"`
}

func NewPort(target, published uint) Port {
	return Port{
		Target:    target,
		Published: published,
		Protocol:  TCP,
		Mode:      "host",
	}
}

type BuildConfig struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
}

type DeployConfig struct {
	RestartPolicy RestartPolicy `yaml:"restart_policy"`
}

type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty"`
	MaxAttempts uint   `yaml:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty"`
}

type IBuildConfigBuilder interface {
	Context(name string) IBuildConfigBuilder
	Docker(name string) IBuildConfigBuilder
	WithArg(key, value string) IBuildConfigBuilder
	Build() BuildConfig
}

type buildConfigBuilder struct {
	context string
	docker  string
	args    map[string]string
}

func (b *buildConfigBuilder) Context(name string) IBuildConfigBuilder {
	b.context = name
	return b
}

func (b *buildConfigBuilder) Docker(name string) IBuildConfigBuilder {
	b.docker = name
	return b
}

func NewBuildConfigBuilder() IBuildConfigBuilder {
	return &buildConfigBuilder{context: ".", docker: "Dockerfile", args: map[string]string{}}
}

func (b *buildConfigBuilder) WithArg(key, value string) IBuildConfigBuilder {
	b.args[key] = value
	return b
}

func (b *buildConfigBuilder) Build() BuildConfig {
	return BuildConfig{
		Context:    b.context,
		Dockerfile: b.docker,
		Args:       b.args,
	}
}

type Service struct {
	Build        BuildConfig            `yaml:"build,omitempty"`
	Image        string                 `yaml:"image,omitempty"`
	Ports        []Port                 `yaml:"ports,omitempty"`
	Volumes      []Volume               `yaml:"volumes,omitempty"`
	Links        []string               `yaml:"links,omitempty"`
	Deploy       DeployConfig           `yaml:"deploy,omitempty"`
	CustomFields map[string]interface{} `yaml:",inline,omitempty"`
}

func NewDockerService(build BuildConfig) Service {
	return Service{Build: build}
}

type MountType string
type BindFlags struct {
	Propagation string `yaml:"propagation"`
}
type VolumeFlags struct {
	NoCopy string `yaml:"nocopy"`
}
type FsSizeConfig struct {
	Size uint `yaml:"size"`
}

type Volume struct {
	MountType MountType    `yaml:"type,omitempty"`
	Source    string       `yaml:"source,omitempty"`
	Target    string       `yaml:"target,omitempty"`
	ReadOnly  bool         `yaml:"readonly,omitempty"`
	Bind      BindFlags    `yaml:"bind,omitempty"`
	Volume    VolumeFlags  `yaml:"volume,omitempty"`
	TempFs    FsSizeConfig `yaml:"tmpfs,omitempty"`
}

type ComposeFile struct {
	Version  string                 `yaml:"version"`
	Services map[string]Service     `yaml:"services"`
	Volumes  map[string]interface{} `yaml:"volumes,omitempty"`
}

func NewComposeFile() *ComposeFile {
	return &ComposeFile{Version: "3", Services: map[string]Service{}}
}

func (c *ComposeFile) AddService(name string, service Service) {
	c.Services[name] = service
}

func (c *ComposeFile) RemoveService(name string) {
	delete(c.Services, name)
}

func (c *ComposeFile) AddVolume(name string) {
	c.Volumes[name] = nil
}

func (c *ComposeFile) RemoveVolume(name string) {
	delete(c.Volumes, name)
}
