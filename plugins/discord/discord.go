package discord

import (
	"sync"

	"code.dopame.me/veonik/squircy3/event"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var ErrNotConnected = errors.New("not connected")

type Config struct {
	Token        string `toml:"token"`
	ActivityName string `toml:"activity"`
	OwnerID      string `toml:"owner"`

	enabled bool
}

type Manager struct {
	conf Config

	ev      *event.Dispatcher
	session *discordgo.Session

	channels map[string]*discordgo.Channel

	mu sync.Mutex
}

func NewManager(ev *event.Dispatcher) *Manager {
	return &Manager{ev: ev, channels: make(map[string]*discordgo.Channel)}
}

func (m *Manager) Configure(c Config) error {
	c.enabled = len(c.Token) > 0
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conf = c
	return nil
}

func (m *Manager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		return errors.New("already connected")
	}
	if !m.conf.enabled {
		return errors.New("not enabled")
	}
	s, err := discordgo.New("Bot " + m.conf.Token)
	if err != nil {
		return err
	}
	m.session = s
	m.session.AddHandler(m.onMessageCreate)
	m.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentGuildMessageTyping | discordgo.IntentsDirectMessages
	m.session.Identify.Presence.Game = discordgo.Activity{Name: m.conf.ActivityName}
	return m.session.Open()
}

func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return ErrNotConnected
	}
	s := m.session
	m.session = nil
	return s.Close()
}

func userToMap(user *discordgo.User) map[string]interface{} {
	return map[string]interface{}{
		"ID":       user.ID,
		"Username": user.Username,
	}
}

func (m *Manager) onMessageCreate(s *discordgo.Session, e *discordgo.MessageCreate) {
	ch, err := m.getChannel(e.ChannelID)
	isDM := false
	if err != nil {
		logrus.Warnf("%s: failed to get channel for %s: %s", PluginName, e.ChannelID, err)
	} else {
		isDM = ch.Type == discordgo.ChannelTypeDM
	}
	m.ev.Emit("discord.MESSAGE", map[string]interface{}{
		"ID":        e.ID,
		"Content":   e.Content,
		"ChannelID": e.ChannelID,
		"GuildID":   e.GuildID,
		"Author":    userToMap(e.Author),
		"FromSelf":  e.Author.ID == s.State.User.ID,
		"IsDM":      isDM,
	})
}

func (m *Manager) MessageChannel(channelID, message string) error {
	_, err := m.session.ChannelMessageSend(channelID, message)
	return err
}

func (m *Manager) MessageChannelTTS(channelID, message string) error {
	_, err := m.session.ChannelMessageSendTTS(channelID, message)
	return err
}

func (m *Manager) CurrentUsername() (string, error) {
	return m.session.State.User.Username, nil
}

func (m *Manager) OwnerID() string {
	return m.conf.OwnerID
}

func (m *Manager) getChannel(id string) (*discordgo.Channel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.channels[id]; ok {
		return ch, nil
	}
	ch, err := m.session.Channel(id)
	if err != nil {
		return nil, err
	}
	m.channels[id] = ch
	return ch, nil
}
