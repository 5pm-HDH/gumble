package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/bontibon/gumble"
)

const responseTemplate = `
<table>
    <tr>
        <td valign="middle">
            <img src='https://www.youtube.com/yt/brand/media/image/YouTube-icon-full_color.png' height="25" />
        </td>
        <td align="center" valign="middle">
            <a href="http://youtu.be/{{ .Data.Id }}">{{ .Data.Title }} ({{ .Data.Duration }})</a>
        </td>
    </tr>
    <tr>
        <td></td>
        <td align="center">
            <a href="http://youtu.be/{{ .Data.Id }}"><img src="{{ .Data.Thumbnail.HqDefault }}" width="250" /></a>
        </td>
    </tr>
</table>`

const linkPattern = `https?://(?:www\.)?(?:youtube\.com/watch\?v=|youtu.be/|youtube.com/v/|youtube.com/v/)([[:alnum:]_\-]+)`

type plugin struct {
	client    *gumble.Client
	keepAlive chan bool
	pattern   *regexp.Regexp
	template  *template.Template
}

func (p *plugin) OnConnect(e *gumble.ConnectEvent) {
	fmt.Printf("youtube-info loaded!\n")
	if pattern, err := regexp.Compile(linkPattern); err != nil {
		panic(err)
	} else {
		p.pattern = pattern
	}
	if template, err := template.New("root").Parse(responseTemplate); err != nil {
		panic(err)
	} else {
		p.template = template
	}
}

func (p *plugin) OnDisconnect(e *gumble.DisconnectEvent) {
}

func (p *plugin) OnTextMessage(e *gumble.TextMessageEvent) {
	if e.Sender == nil {
		return
	}
	matches := p.pattern.FindStringSubmatch(e.Message)
	if len(matches) != 2 {
		return
	}
	go fetchYoutubeInfo(p, matches[1])
}

type videoInfo struct {
	Data struct {
		Id        string
		Title     string
		Duration  time.Duration
		Thumbnail struct {
			HqDefault string
		}
	}
}

func fetchYoutubeInfo(p *plugin, id string) {
	var info videoInfo

	// Fetch + parse video info
	url := fmt.Sprintf("http://gdata.youtube.com/feeds/api/videos/%s?v=2&alt=jsonc", id)
	if resp, err := http.Get(url); err != nil {
		return
	} else {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&info); err != nil {
			return
		}
		info.Data.Duration *= time.Second
		resp.Body.Close()
	}

	// Create response string
	var buffer bytes.Buffer
	if err := p.template.Execute(&buffer, info); err != nil {
		return
	}
	message := gumble.TextMessage{
		Channels: []*gumble.Channel{
			p.client.Self().Channel(),
		},
		Message: buffer.String(),
	}
	p.client.Send(&message)
}

func (p *plugin) OnUserChange(e *gumble.UserChangeEvent) {
}

func (p *plugin) OnChannelChange(e *gumble.ChannelChangeEvent) {
}

func main() {
	// flags
	server := flag.String("server", "localhost:64738", "mumble server address")
	username := flag.String("username", "youtube-info-bot", "client username")
	password := flag.String("password", "", "client password")
	insecure := flag.Bool("insecure", false, "skip checking server certificate")

	flag.Parse()

	// implementation
	p := plugin{
		keepAlive: make(chan bool),
	}

	// client
	p.client = gumble.NewClient()
	if *insecure {
		p.client.TlsConfig().InsecureSkipVerify = true
	}
	p.client.Attach(&p)
	if err := p.client.Dial(*username, *password, *server); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	<-p.keepAlive
}