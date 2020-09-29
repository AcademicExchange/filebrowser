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
	cfgFile  string
	env      string
	file     string
	dir      string
	isReload bool
	tmp      string = ".tmp.gob"
	svrMap   map[string]Server
)

var log = logging.MustGetLogger("example")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000}|%{longfile}|%{level:.6s}|%{color:reset}%{message}`,
)

func initLogger() {
	backend := logging.NewLogBackend(os.Stdout, "", 0)

	backendFormatter := logging.NewBackendFormatter(backend, format)

	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.ERROR, "")

	logging.SetBackend(backendLeveled, backendFormatter)
}

// login to target node or tcm
func isLoginCompleted(env string, st *Store, so *Socket) bool {
	var jwt string
	var uid string
	if jwt = st.GetJwt(env, so.GetUrl()); jwt == "" { // jwt is null, need login
		status, body := so.login()
		if status != 200 {
			log.Errorf("login %s failed for %s", so.GetUrl(), body)
			return false
		}
		jwt = body
		st.SetJwt(env, so.GetUrl(), jwt)
	}
	// check whether jwt is still valid
	if status, _ := so.renew(jwt); status != 200 {
		if status == 403 { // jwt expired, need relogin
			status, body := so.login()
			if status != 200 {
				log.Errorf("login %s failed for %s", so.GetUrl(), body)
				return false
			}
			jwt = body
			st.SetJwt(env, so.GetUrl(), jwt)
		}
	}
	// log.Debugf("jwt: %s", jwt)

	if uid = st.GetUuid(env, so.GetUrl()); uid == "" {
		uid = uuid.NewV4().String()
		st.SetUuid(env, so.GetUrl(), uid)
	}
	// log.Debugf("uuid: %s", uid)
	return true
}

// this function can only be called after successful login
func isUploadCompleted(env string, st *Store, so *Socket, file string, dir string) bool {
	jwt := st.GetJwt(env, so.GetUrl())
	uid := st.GetUuid(env, so.GetUrl())
	status, body := so.upload(file, dir, uid, jwt)
	if status != 200 {
		log.Errorf("upload file %s failed for %s", file, body)
		return false
	}
	log.Infof("upload file %s succeed", file)
	return true
}

// this function can only be called by tcm after the file is uploaded successfully
func isReloadCompleted(env string, st *Store, tcm *Socket) bool {
    bRet := false
    jwt := st.GetJwt(env, tcm.GetUrl())
	uid := st.GetUuid(env, tcm.GetUrl())
	status, body := tcm.reload(uid, jwt)
	if status != 200 {
		log.Errorf("reload status: %d", status)
		log.Errorf("reload result: %s", body)
	} else {
		log.Infof("reload status: %d", status)
        log.Infof("reload result: %s", body)
        bRet = true
	}
    st.SetUuid(env, tcm.GetUrl(), "") // reload is complete, reset uuid
    return bRet
}

var rootCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload configuration files via HTTP and reload configuration in target environment",
	Long: `upload configuration files via HTTP and reload config in target environment. Set the flags for the options
you want to change. Other options will remain unchanged. `,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if !isReload {
			if file == "" || dir == "" {
				cmd.Help()
				os.Exit(1)
			}
		}
		svr, found := svrMap[env]
		if !found {
			log.Errorf("env: %s not found", env)
			cmd.Help()
			os.Exit(1)
		}
		tcm := svr.Tcm
		nodes := svr.Nodes

		s := NewStore(tmp)
		defer s.Save()

		// login tcm and all nodes
		if ok := isLoginCompleted(env, s, &tcm); !ok {
            log.Errorf("login tcm %v failed", tcm)
            os.Exit(1)
		}
		for _, node := range nodes {
			if ok := isLoginCompleted(env, s, &node); !ok {
                log.Errorf("login node %v failed", node)
                os.Exit(1)
			}
		}

		// upload file to all nodes
		if file != "" && dir != "" {
			if !(strings.Contains(file, "ClientConfig") || strings.Contains(file, "Common") || strings.Contains(file, "ServerConfig")) {
                log.Errorf("only files in the ClientConfig or ServerConfig directory can be uploaded")
                os.Exit(1)
			}

			if !(strings.HasPrefix(dir, "/wedo/ClientConfig") || strings.HasPrefix(dir, "/wedo/ServerConfig")) {
                log.Errorf("the relative path can only start with /wedo/ClientConfig or /wedo/ServerConfig")
                os.Exit(1)
			}

			for _, node := range nodes {
				if ok := isUploadCompleted(env, s, &node, file, dir); !ok {
                    log.Errorf("upload file %s to node %v failed", file, node)
                    os.Exit(1)
				}
			}
		}

		// only tcm can reload config
		if isReload {
            if ok := isReloadCompleted(env, s, &tcm); !ok {
                os.Exit(1)
            }
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
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}
	svrMap = cfg
	rootCmd.Flags().StringVarP(&env, "env", "e", "dailybuild", "the name of target environment")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "the absolute path of the configuration file to be uploaded")
	rootCmd.Flags().StringVarP(&dir, "dir", "d", "", `the relative path of the uploaded configuration file or the uploaded configuration directory 
which starts with "/wedo/ClientConfig" or "/wedo/ServerConfig"`)
	rootCmd.Flags().BoolVarP(&isReload, "reload", "r", false, "whether to reload configurations")

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
