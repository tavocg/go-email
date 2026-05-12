package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

// Message holds the fields needed to render a MIME email.
type Message struct {
	From    string
	To      []string
	Cc      []string
	Bcc     []string
	Subject string

	plainBody string
	htmlBody  string

	attachments []Attachment
}

// Attachment stores the bytes and metadata needed for a MIME attachment.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// NewPlainMessage builds a text/plain email.
func NewPlainMessage(from string, to []string, subject, body string) *Message {
	return &Message{
		From:      from,
		To:        append([]string(nil), to...),
		Subject:   subject,
		plainBody: body,
	}
}

// NewHTMLMessage builds a text/html email.
func NewHTMLMessage(from string, to []string, subject, body string) *Message {
	return &Message{
		From:     from,
		To:       append([]string(nil), to...),
		Subject:  subject,
		htmlBody: body,
	}
}

// NewAlternativeMessage builds a multipart/alternative email with both plain
// text and HTML bodies.
func NewAlternativeMessage(from string, to []string, subject, plainBody, htmlBody string) *Message {
	return &Message{
		From:      from,
		To:        append([]string(nil), to...),
		Subject:   subject,
		plainBody: plainBody,
		htmlBody:  htmlBody,
	}
}

// Attach stores an attachment and infers a content type from its bytes or
// filename.
func (m *Message) Attach(filename string, content []byte) {
	contentType := http.DetectContentType(content)
	if contentType == "application/octet-stream" {
		if inferred := mime.TypeByExtension(filepath.Ext(filename)); inferred != "" {
			contentType = inferred
		}
	}

	m.attachments = append(m.attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	})
}

// AttachWithType stores an attachment with an explicit content type.
func (m *Message) AttachWithType(filename, contentType string, content []byte) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	m.attachments = append(m.attachments, Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	})
}

func (m *Message) recipients() []string {
	recipients := make([]string, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.Cc...)
	recipients = append(recipients, m.Bcc...)
	return recipients
}

func (m *Message) bytes() ([]byte, error) {
	if m == nil {
		return nil, ErrNilMessage
	}
	if strings.TrimSpace(m.From) == "" {
		return nil, ErrMessageMissingFrom
	}
	if len(m.recipients()) == 0 {
		return nil, ErrMissingRecipients
	}
	if m.plainBody == "" && m.htmlBody == "" && len(m.attachments) == 0 {
		return nil, ErrMessageMissingBody
	}

	contentType, transferEncoding, body, err := m.buildBody()
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writeHeader(&buffer, "From", m.From)
	writeHeader(&buffer, "To", strings.Join(m.To, ", "))
	if len(m.Cc) > 0 {
		writeHeader(&buffer, "Cc", strings.Join(m.Cc, ", "))
	}
	writeHeader(&buffer, "Subject", encodeHeader(m.Subject))
	writeHeader(&buffer, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&buffer, "MIME-Version", "1.0")
	writeHeader(&buffer, "Content-Type", contentType)
	if transferEncoding != "" {
		writeHeader(&buffer, "Content-Transfer-Encoding", transferEncoding)
	}
	buffer.WriteString("\r\n")
	buffer.Write(body)
	return buffer.Bytes(), nil
}

func (m *Message) buildBody() (string, string, []byte, error) {
	switch {
	case len(m.attachments) > 0:
		return m.buildMixedBody()
	case m.plainBody != "" && m.htmlBody != "":
		return m.buildAlternativeBody()
	case m.htmlBody != "":
		return "text/html; charset=UTF-8", "quoted-printable", encodeTextBody(m.htmlBody), nil
	default:
		return "text/plain; charset=UTF-8", "quoted-printable", encodeTextBody(m.plainBody), nil
	}
}

func (m *Message) buildMixedBody() (string, string, []byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	if err := m.writeInlineBody(writer); err != nil {
		return "", "", nil, err
	}

	for _, attachment := range m.attachments {
		headers := textHeaders(
			"Content-Type", attachmentMediaType(attachment),
			"Content-Disposition", formatMediaType("attachment", map[string]string{"filename": attachment.Filename}),
			"Content-Transfer-Encoding", "base64",
		)

		part, err := writer.CreatePart(headers)
		if err != nil {
			return "", "", nil, err
		}
		if err := writeBase64(part, attachment.Content); err != nil {
			return "", "", nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return "", "", nil, err
	}

	return fmt.Sprintf("multipart/mixed; boundary=%q", writer.Boundary()), "", buffer.Bytes(), nil
}

func (m *Message) buildAlternativeBody() (string, string, []byte, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	if err := writeTextPart(writer, "plain", m.plainBody); err != nil {
		return "", "", nil, err
	}
	if err := writeTextPart(writer, "html", m.htmlBody); err != nil {
		return "", "", nil, err
	}
	if err := writer.Close(); err != nil {
		return "", "", nil, err
	}

	return fmt.Sprintf("multipart/alternative; boundary=%q", writer.Boundary()), "", buffer.Bytes(), nil
}

func (m *Message) writeInlineBody(writer *multipart.Writer) error {
	switch {
	case m.plainBody != "" && m.htmlBody != "":
		var body bytes.Buffer
		alternative := multipart.NewWriter(&body)

		if err := writeTextPart(alternative, "plain", m.plainBody); err != nil {
			return err
		}
		if err := writeTextPart(alternative, "html", m.htmlBody); err != nil {
			return err
		}
		if err := alternative.Close(); err != nil {
			return err
		}

		headers := textHeaders(
			"Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", alternative.Boundary()),
		)
		part, err := writer.CreatePart(headers)
		if err != nil {
			return err
		}
		_, err = part.Write(body.Bytes())
		return err
	case m.htmlBody != "":
		return writeTextPart(writer, "html", m.htmlBody)
	case m.plainBody != "":
		return writeTextPart(writer, "plain", m.plainBody)
	default:
		return nil
	}
}

func writeTextPart(writer *multipart.Writer, subtype, body string) error {
	headers := textHeaders(
		"Content-Type", fmt.Sprintf("text/%s; charset=UTF-8", subtype),
		"Content-Transfer-Encoding", "quoted-printable",
	)

	part, err := writer.CreatePart(headers)
	if err != nil {
		return err
	}

	_, err = part.Write(encodeTextBody(body))
	return err
}

func writeBase64(writer io.Writer, content []byte) error {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
	base64.StdEncoding.Encode(encoded, content)

	for len(encoded) > 76 {
		if _, err := writer.Write(encoded[:76]); err != nil {
			return err
		}
		if _, err := writer.Write([]byte("\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}

	if len(encoded) > 0 {
		if _, err := writer.Write(encoded); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte("\r\n"))
	return err
}

func encodeTextBody(body string) []byte {
	var buffer bytes.Buffer
	writer := quotedprintable.NewWriter(&buffer)
	_, _ = writer.Write([]byte(body))
	_ = writer.Close()
	return buffer.Bytes()
}

func writeHeader(buffer *bytes.Buffer, key, value string) {
	if value == "" {
		return
	}
	buffer.WriteString(key)
	buffer.WriteString(": ")
	buffer.WriteString(value)
	buffer.WriteString("\r\n")
}

func encodeHeader(value string) string {
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("utf-8", value)
}

func isASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}

func formatMediaType(mediaType string, params map[string]string) string {
	return mime.FormatMediaType(mediaType, params)
}

func attachmentMediaType(attachment Attachment) string {
	mediaType := attachment.ContentType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	parsedType, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return mediaType
	}

	if params == nil {
		params = make(map[string]string, 1)
	}
	params["name"] = attachment.Filename
	return mime.FormatMediaType(parsedType, params)
}

func textHeaders(values ...string) textproto.MIMEHeader {
	headers := make(textproto.MIMEHeader, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		headers.Set(values[index], values[index+1])
	}
	return headers
}
