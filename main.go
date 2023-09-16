package main

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
	"strconv"

	"encoding/json"
	"text/template"
	"time"
	"os"

	"fmt"
	"io/ioutil"
	"path/filepath"

	"net"
	"net/http"
	"net/smtp"
	"net/url"

	"github.com/romana/rlog"
	"github.com/ghodss/yaml"
	"github.com/antonholmquist/jason"

)
var config string
func MustNoErr(e error, args ...interface{}) {
	if e != nil {
		if args != nil && args[0] != nil {
			rlog.Error(args[0])
		}
		rlog.Critical(e.Error())
		panic(e.Error())
	}
}
func NilOnErr(e error, args ...interface{}) {
	if e != nil {
		if args != nil && args[0] != nil {
			rlog.Error(args[0])
		}
		rlog.Critical(e.Error())
		panic(e.Error())
	}
}
func LogDebugNoErr(e error, args ...interface{}) {
	if e != nil {
		if args != nil && args[0] != nil {
			rlog.Debug(args[0])
		}
		rlog.Debug(e.Error())
	}
}
func OnlyLogError(e error, args ...interface{}) {
	if e != nil {
		if args != nil && args[0] != nil {
			rlog.Error(args[0])
		}
		rlog.Error(e.Error())
	}
}

func MustString(v string, e error) string {
	MustNoErr(e)
	return v
}
func MaybeString(v string, e error) string {
	return v
}
func MustByteArray(v []byte, e error) []byte {
	MustNoErr(e)
	return v
}

func LoadConfig() {
	config = string(
		MustByteArray(yaml.YAMLToJSON(
			MustByteArray(ioutil.ReadFile(
				MustString(filepath.Abs("./config.yml")))))))
}

func MatchEvent(eventO *jason.Object, notification *jason.Object, regex bool) bool {
	var allMatched bool
	allMatched = true
	var toMatchKey string
	toMatchKey = "when"
	if regex {
		toMatchKey = "when_regex"
	}
	checkIfRegexMatches, e := notification.GetObject(toMatchKey)
	if e != nil {
		rlog.Debugf("No %s key to test, returning true", toMatchKey)
		return true
	}

	for eventKey, regexStrings := range checkIfRegexMatches.Map() {
		regexStrings, e := regexStrings.Array()
		MustNoErr(e)
		for _, regexString := range regexStrings {

			regexString, e := regexString.String()
			MustNoErr(e)
			eventKeyAsArray := strings.Split(eventKey, ".")

			eventValue, e := eventO.GetString(eventKeyAsArray...)
			if e != nil {
				return false
			}
			var matched bool

			if regex {
				matched, e = regexp.Match(regexString, []byte(eventValue))
				MustNoErr(e)
			} else {
				matched = regexString == eventValue
			}
			allMatched = allMatched && matched
			rlog.Debugf("matched %v [%s] %s == %s \n", matched, eventKey, eventValue, regexString)

		}
	}
	return allMatched

}

func CheckAndNotify(event string) {
	rlog.Debugf("Event: %s", event)
	eventO, e := jason.NewObjectFromBytes([]byte(event))
	LogDebugNoErr(e, "Not valid json event")
	if e != nil {
		return
	}

	configO, e := jason.NewObjectFromBytes([]byte(config))
	MustNoErr(e, "Not valid config")
	notifications, e := configO.GetObjectArray("notifications")
	MustNoErr(e, "No notifications defined in config")

	for _, notification := range notifications {
		title, e := notification.GetString("title")
		MustNoErr(e, "No Title defined for notification")
		rlog.Debugf("Checking \"%s\"\n", title)
		if MatchEvent(eventO, notification, true) && MatchEvent(eventO, notification, false) {
			rlog.Debugf("Triggering \"%s\"\n", title)
			notify, e := notification.GetStringArray("notify")
			MustNoErr(e, "No notify key defined for notification")

			PrepareAndSendNotifications(title, event, notify)
		}
	}
}
func TimeStampFormat (timestamp interface{}, format string) string {
	timestampF, _ := strconv.ParseFloat(fmt.Sprintf("%f", timestamp), 64)
	return fmt.Sprintf("%v", time.Unix(int64(timestampF), 0).Format(format))
}
func SendNotification(notification string, eventO *map[string]interface{}) {
	configO, e := jason.NewObjectFromBytes([]byte(config))
	MustNoErr(e, "Not valid config")
	var templateFile = MustString(configO.GetString("notifiers", notification, "template"))
	var dataEncoding = MaybeString(configO.GetString("notifiers", notification, "data_encoding"))


	var tmpl = template.Must(template.New(filepath.Base(templateFile)).
		Funcs(template.FuncMap{"TimeStampFormat": TimeStampFormat}).
		ParseFiles(MustString(filepath.Abs(templateFile))))
	var output bytes.Buffer      // will contain output to send
	e = tmpl.Execute(&output, &eventO)
	MustNoErr(e, fmt.Sprintf("Problems parsing notification template: %s", templateFile))
	var notificationUrl = MustString(configO.GetString("notifiers", notification, "url"))
	rlog.Debugf("ALERT_TO %v", notificationUrl)
	rlog.Debugf("ALERT_TEMPLATE %v", output.String())


	urlParsed, e := url.Parse(notificationUrl)
	OnlyLogError(e)
	rlog.Debug(urlParsed.Scheme)
	if urlParsed.Scheme == "smtp" || urlParsed.Scheme == "smtps" {
		sendNotificationViaSmtp(urlParsed, output.String(), urlParsed.Scheme == "smtps")
	} else {
		sendNotificationViaHttp(urlParsed, output.String(), dataEncoding)
	}

}

func sendNotificationViaSmtp(notificatonUrl *url.URL, message string, verifyTls bool) {
	var auth = smtp.Auth(nil)
	password, passSet := notificatonUrl.User.Password()
	if passSet != false  {
		auth = smtp.PlainAuth("", notificatonUrl.User.Username(), password, notificatonUrl.Hostname())
	}

	smtpQueryString := notificatonUrl.Query()
	err := smtp.SendMail(notificatonUrl.Host, auth, smtpQueryString["from"][0], smtpQueryString["to"], []byte(message))
	OnlyLogError(err)
}
func sendNotificationViaHttp(notificationUrl *url.URL, message string, dataEncoding string) {
    var resp *http.Response
    var err error

    switch {
    case strings.HasPrefix(dataEncoding, "urlencode."):
        parts := strings.SplitN(dataEncoding, ".", 2)
        if len(parts) != 2 {
            rlog.Warnf("Skipping notification > Got unsupported data_encoding ( %v )", dataEncoding)
            return
        }
        payloadKey := parts[1]
        values := url.Values{payloadKey: {message}}
        resp, err = http.PostForm(notificationUrl.String(), values)
    case dataEncoding == "json":
        resp, err = http.Post(notificationUrl.String(), "application/json", strings.NewReader(message))
    default:
        rlog.Warnf("Skipping notification > Got unsupported data_encoding ( %v )", dataEncoding)
        return
    }

    OnlyLogError(err)

    if resp != nil && resp.StatusCode != http.StatusOK {
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        OnlyLogError(err, "Error posting notification")
        rlog.Warnf("Didn't receive 200 OK when posting notification %v, %v", resp.Status, string(body))
    }
}
func PrepareAndSendNotifications (title string, event string, notifications []string) {

	rlog.Infof("notify %s %v\n", title, notifications)

	var eventO = map[string]interface{}{}
	e := json.Unmarshal([]byte(event), &eventO)
	MustNoErr(e)
	eventO["dockerHostLabel"] = os.Getenv("DOCKER_HOST_LABEL")
	eventO["eventJSON"] = string(MustByteArray(json.MarshalIndent(eventO, "", "  ")))
	eventO["notificationTitle"] = title

	for _, notification := range notifications {
		SendNotification(notification, &eventO)
	}
}

func main() {

	// TODO: allow http(s) with tls and all
	conn, e := net.Dial("unix", "/var/run/docker.sock")
	MustNoErr(e)

	LoadConfig()
	configO, e := jason.NewObjectFromBytes([]byte(config))
	MustNoErr(e)
	var filters = ""
	filtersO, e := configO.GetObject("filters")
	if e == nil {
		filters = filtersO.String()
	}

	fmt.Fprintf(
		conn,
		"GET /v%s/events?filters=%s HTTP/1.0\r\n\r\n",
		os.Getenv("DOCKER_API_VERSION"),
		filters)

	reader := bufio.NewReader(conn)
	for {
		CheckAndNotify(string(MustByteArray(reader.ReadBytes('\n'))))
	}
	rlog.Warnf("Fin - %v", time.Now())
}
