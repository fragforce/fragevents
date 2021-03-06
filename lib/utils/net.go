package utils

import (
	"context"
	"encoding/json"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"os"
)

type WanIP34 struct {
	IP      string `json:"ip"`
	Message string `json:"message"`
}

func init() {
	viper.SetDefault("wan.iplookup", "https://www.3-4.us/") // Owned by Paulson, Heroku hosted
}

//GetLocalIP gets the first non-loopback interface IP
func GetLocalIP() (string, error) { // https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
	//addrs, err := net.InterfaceAddrs()
	//if err != nil {
	//	return "", err
	//}
	//for _, address := range addrs {
	//	// check the address type and if it is not a loopback the display it
	//	if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !strings.HasPrefix(address.String(), "172.17.") {
	//		if ipnet.IP.To4() != nil {
	//			return ipnet.IP.String(), nil
	//		}
	//	}
	//}
	//return "", errors.New("no IPs found")
	// FIXME: Error Checking!
	return os.Getenv("HEROKU_PRIVATE_IP"), nil
}

func GetLocalNodeFQDN() (string, error) { // https://devcenter.heroku.com/articles/dyno-dns-service-discovery
	return os.Getenv("HEROKU_DNS_DYNO_NAME"), nil
}

//GetExternalIP uses an external website to fetch our WAN IP
func GetExternalIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", viper.GetString("wan.iplookup"), nil)
	if err != nil {
		return "", err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var wanIP WanIP34
	if err := json.Unmarshal(body, &wanIP); err != nil {
		return "", err
	}

	return wanIP.IP, nil
}
