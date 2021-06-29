package messages_helpers

import tb "gopkg.in/tucnak/telebot.v2"

func TextFromMsg(m *tb.Message) string {
	text := m.Text
	if text == "" {
		text = m.Caption
	}
	return text
}
