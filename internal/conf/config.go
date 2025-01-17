package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Configure struct {
	Key                 string
	LocalSSHAddr        string
	LocalListenAddr     string
	FullNode            bool
	ID                  string
	SignalingServerAddr string
	RTCConf             webrtc.Configuration
}

type ConfManager struct {
	Conf  *Configure
	Viper *viper.Viper
	Path  string
}

var defaultConfig = Configure{
	LocalListenAddr:     "127.0.0.1:2222",
	LocalSSHAddr:        "127.0.0.1:22",
	FullNode:            true,
	ID:                  uuid.New().String(),
	Key:                 uuid.New().String(),
	SignalingServerAddr: "http://peer1.cotnetwork.com:8990",
	RTCConf: webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
					"stun:stun1.l.google.com:19302",
					"stun:stun2.l.google.com:19302",
					"stun:stun3.l.google.com:19302",
					"stun:stun4.l.google.com:19302",
				},
			},
		},
	},
}

func clearKnownHosts(subStr string) {
	subStr = strings.Replace(subStr, "127.0.0.1", "[127.0.0.1]", 1)
	//[127.0.0.1]:2222
	fileName := os.Getenv("HOME") + "/.ssh/known_hosts"
	input, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println(err)
		return
	}
	lines := strings.Split(string(input), "\n")
	var newLines []string
	for i, line := range lines {
		if strings.Contains(line, subStr) {
		} else {
			newLines = append(newLines, lines[i])
		}
	}
	output := strings.Join(newLines, "\n")
	err = ioutil.WriteFile(fileName, []byte(output), 0644)
	if err != nil {
		log.Println(err)
		return
	}
	//ioutil.WriteFile(fileName, []byte(res), 544)
}

func NewConfManager(path string) *ConfManager {
	var tmp Configure
	vp := viper.New()
	vp.SetConfigName(".sshx_config")
	vp.SetConfigType("json")
	vp.AddConfigPath(path)
	vp.WatchConfig()
	vp.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		err := vp.Unmarshal(&tmp)
		if err != nil {
			panic(err)
		}
		clearKnownHosts(tmp.LocalListenAddr)
	})
	err := vp.ReadInConfig() // Find and read the config file
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			bs, _ := json.MarshalIndent(defaultConfig, "", "  ")
			vp.ReadConfig(bytes.NewBuffer(bs))
			log.Print("Write config ...\n", string(bs))
			err = vp.WriteConfigAs(path + "/.sshx_config.json")
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	err = vp.Unmarshal(&tmp)
	if err != nil {
		panic(err)
	}
	//log.Println(tmp)
	clearKnownHosts(tmp.LocalListenAddr)
	return &ConfManager{
		Conf:  &tmp,
		Viper: vp,
		Path:  path,
	}
}

func (cm *ConfManager) Set(key, value string) {
	clearKnownHosts(cm.Conf.LocalListenAddr)
	cm.Viper.Set(key, value)
	log.Print("Write config ...")
	err := cm.Viper.Unmarshal(cm.Conf)
	if err != nil {
		panic(err)
	}
	cm.Viper.WriteConfigAs(cm.Path + "/.sshx_config.json")

}

func (cm *ConfManager) Show() {
	bs, _ := json.MarshalIndent(cm.Conf, "", "  ")
	log.Println(string(bs))
}
