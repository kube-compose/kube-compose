package cmd

import (
	"github.com/jbrekelmans/kube-compose/pkg/config"
<<<<<<< HEAD
	"github.com/spf13/cobra"
=======
	"github.com/urfave/cli"

	// gcp plugin doesn't provide any functions therefore importing as "_" is fine.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
>>>>>>> adf01d6... Start independent services in kube-compose defined in docker-compose.yml (#49)

	// Plugin does not export any functions therefore it is ignored IE. "_"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

<<<<<<< HEAD
func setFromKubeConfig(cfg *config.Config) error {
=======
const (
	environmentIDFlagName = "env-id"
	namespaceFlagName     = "namespace"
)

func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   environmentIDFlagName + ", e",
			EnvVar: "KUBECOMPOSE_ENVID",
			Usage: "used to isolate environments deployed to a shared namespace, " +
				"by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors",
		},
		cli.StringFlag{
			Name:   namespaceFlagName + ", n",
			EnvVar: "KUBECOMPOSE_NAMESPACE",
			Usage:  "the target Kubernetes namespace",
		},
	}
}

func newConfigFromEnv() (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
>>>>>>> 63e2104... Fixing lint issues
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, &overrides)
	kubeConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	cfg.KubeConfig = kubeConfig
	cfg.Namespace = namespace
	return nil
}

<<<<<<< HEAD
func getFileFlag(cmd *cobra.Command) (*string, error) {
	var file *string
	if cmd.Flags().Changed("file") {
		fileStr, err := cmd.Flags().GetString("file")
		if err != nil {
			return nil, err
		}
		file = new(string)
		*file = fileStr
=======
func updateConfigFromCli(cfg *config.Config, c *cli.Context) error {
	environmentID := c.GlobalString(environmentIDFlagName)
	cfg.Services = c.Args()
	if environmentID == "" && !c.GlobalIsSet(environmentIDFlagName) {
		return fmt.Errorf("the environment id is required")
	} else if environmentID == "" {
		return fmt.Errorf("environment id must not be empty")
>>>>>>> adf01d6... Start independent services in kube-compose defined in docker-compose.yml (#49)
	}
	return file, nil
}

<<<<<<< HEAD
func upOrDownCommandCommon(cmd *cobra.Command, args []string) (*config.Config, error) {
	file, err := getFileFlag(cmd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.New(file)
	if err != nil {
		return nil, err
	}
	err = setFromKubeConfig(cfg)
	if err != nil {
		return nil, err
	}
	cfg.EnvironmentID, _ = cmd.Flags().GetString("env-id")
	if namespace, _ := cmd.Flags().GetString("namespace"); namespace != "" {
=======
	namespace := c.GlobalString(namespaceFlagName)
	if len(namespace) > 0 || c.GlobalIsSet(namespaceFlagName) {
		if namespace == "" {
			return fmt.Errorf("namespace must not be empty")
		}
>>>>>>> 63e2104... Fixing lint issues
		cfg.Namespace = namespace
	}
	if len(args) == 0 {
		cfg.SetFilterToMatchAll()
	} else {
		err = cfg.SetFilter(args)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
