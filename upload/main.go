package main

import (
	"fmt"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	env     string
	file    string
	dir     string
	reload  bool
	tmp     string = ".tmp.gob"
	host    string = "http://idc.iwedo.oa.com/idcnet/file_browser/"
)

var envMap = map[string]string{
	"dailybuild": "57445",
	"dailyidc":   "57446",
	"test":       "57544",
	"lighttest":  "57644",
	"shenhe":     "57744",
	"banhao":     "57844",
	"ce":         "57944",
	"pressure":   "64344",
}

var log = logging.MustGetLogger("example")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000}|%{longfile}|%{level:.6s}|%{color:reset}%{message}`,
)

func initLogger() {
	backend1 := logging.NewLogBackend(os.Stdout, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)

	backend2Formatter := logging.NewBackendFormatter(backend2, format)

	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")

	logging.SetBackend(backend1Leveled, backend2Formatter)
}

var rootCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload configuration files via HTTP and reload configuration in target environment",
	Long: `upload configuration files via HTTP and reload config in target environment. Set the flags for the options
you want to change. Other options will remain unchanged. `,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		host += envMap[env]
		var jwt string
		var uid string
		s := NewStore(tmp)
		defer s.Save()

		if jwt = s.GetJWT(); jwt == "" { // jwt is null, need login
			status, body := login("wedo", "wedo")
			if status != 200 {
				log.Errorf("login failed")
				return
			}
			jwt = body
			s.SetJWT(jwt)
		}

		if status, _ := renew(jwt); status != 200 {
			if status == 403 { // jwt expired, need relogin
				status, body := login("wedo", "wedo")
				if status != 200 {
					log.Errorf("login failed")
					return
				}
				jwt = body
				s.SetJWT(jwt)
			}
		}
		log.Debugf("jwt: %s", jwt)

		if uid = s.GetUUID(); uid == "" {
			uid = uuid.NewV4().String()
			s.SetUUID(uid)
		}
		log.Debugf("uuid: %s", uid)

		if file != "" && dir != "" {
			if !(strings.Contains(file, "ClientConfig") || strings.Contains(file, "ServerConfig")) {
				log.Errorf("only files in the ClientConfig or ServerConfig directory can be uploaded")
				return
			}

			if !(strings.HasPrefix(dir, "/wedo/ClientConfig") || strings.HasPrefix(dir, "/wedo/ServerConfig")) {
				log.Errorf("the relative path can only start with /wedo/ClientConfig or /wedo/ServerConfig")
				return
			}

			status, body := upload(file, dir, uid, jwt)
			if status != 200 {
				log.Errorf("upload file %s failed for %s", file, body)
				return
			}
			log.Infof("upload file %s succeed", file)
		}

		if reload {
			status, body := reloadConfig(uid, jwt)
			if status != 200 {
				log.Errorf("reload status: %d", status)
				log.Errorf("reload result: %s", body)
			} else {
				log.Infof("reload status: %d", status)
				log.Infof("reload result: %s", body)
			}
			s.SetUUID("") // reload is complete, reset uuid
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
    initLogger()
	rootCmd.Flags().StringVarP(&env, "env", "e", "dailybuild", "the name of target environment")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "the absolute path of the configuration file to be uploaded")
	rootCmd.Flags().StringVarP(&dir, "dir", "d", "", `the relative path of the uploaded configuration file or the uploaded configuration directory 
which starts with "/wedo/ClientConfig" or "/wedo/ServerConfig"`)
	rootCmd.Flags().BoolVarP(&reload, "reload", "r", false, "whether to reload configurations")

}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".demo")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	Execute()
}
