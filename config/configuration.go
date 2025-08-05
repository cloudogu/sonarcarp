package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultFileName = "carp.yml"

type Configuration struct {
	BaseUrl                            string `yaml:"base-url"`
	CasUrl                             string `yaml:"cas-url"`
	ServiceUrl                         string `yaml:"service-url"`
	Target                             string `yaml:"target-url"`
	ResourcePath                       string `yaml:"resource-path"`
	SkipSSLVerification                bool   `yaml:"skip-ssl-verification"`
	Port                               int    `yaml:"port"`
	PrincipalHeader                    string `yaml:"principal-header"`
	RoleHeader                         string `yaml:"role-header"`
	MailHeader                         string `yaml:"mail-header"`
	NameHeader                         string `yaml:"name-header"`
	LogoutMethod                       string `yaml:"logout-method"`
	LogoutPath                         string `yaml:"logout-path"`
	ForwardUnauthenticatedRESTRequests bool   `yaml:"forward-unauthenticated-rest-requests"`
	LoggingFormat                      string `yaml:"log-format"`
	LogLevel                           string `yaml:"log-level"`
	CreateUserEndpoint                 string `yaml:"create-user-endpoint"`
	CreateGroupEndpoint                string `yaml:"create-group-endpoint"`
	GetUserGroupsEndpoint              string `yaml:"get-user-groups-endpoint"`
	GetUserEndpoint                    string `yaml:"get-user-endpoint"`
	AddUserToGroupEndpoint             string `yaml:"add-user-to-group-endpoint"`
	RemoveUserFromGroupEndpoint        string `yaml:"remove-user-from-group-endpoint"`
	SearchGroupByNameEndpoint          string `yaml:"search-team-by-name-endpoint"`
	SetOrganizationRoleEndpoint        string `yaml:"set-organization-role-endpoint"`
	CesAdminGroup                      string `yaml:"ces-admin-group"`
	GrafanaAdminGroup                  string `yaml:"grafana-admin-group"`
	GrafanaWriterGroup                 string `yaml:"grafana-writer-group"`
	GrafanaReaderGroup                 string `yaml:"grafana-reader-group"`
	ApplicationExecCommand             string `yaml:"application-exec-command"`
	CarpResourcePath                   string `yaml:"carp-resource-path"`
}

func InitializeAndReadConfiguration() (Configuration, error) {
	configuration, err := readConfiguration()
	if err != nil {
		return Configuration{}, fmt.Errorf("could not read configuration: %w", err)
	}

	err = initLogger(configuration)
	if err != nil {
		return Configuration{}, fmt.Errorf("could not configure logger: %w", err)
	}

	return configuration, nil
}

func readConfiguration() (Configuration, error) {
	confPath := defaultFileName

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}

			if !isYamlFile(arg) {
				log.Warningf("Provided config file %s is no yaml file, try to use default file %s", arg, defaultFileName)
				break
			}

			confPath = arg
		}
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to read file from path %s: %w", confPath, err)
	}

	var config Configuration

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to unmarshal file to configuration: %w", err)
	}

	return config, nil
}

func isYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}
