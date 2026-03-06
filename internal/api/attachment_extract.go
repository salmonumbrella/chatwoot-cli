package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

const (
	DefaultDocumentExtractLimit         = 3
	DefaultDocumentExtractMaxBytes      = int64(10 * 1024 * 1024)
	DefaultDocumentExtractMaxTotalBytes = int64(25 * 1024 * 1024)
	DefaultDocumentExtractMaxChars      = 12000
)

type ConversationAttachmentExtractOptions struct {
	Indexes           []int
	Limit             int
	MaxBytes          int64
	MaxTotalBytes     int64
	MaxChars          int
	UnsafeNoSizeLimit bool
}

type ConversationAttachmentExtractResult struct {
	ConversationID int                        `json:"conversation_id"`
	Items          []ExtractedAttachmentText  `json:"items"`
	Meta           ConversationAttachmentMeta `json:"meta"`
}

type ConversationAttachmentMeta struct {
	TotalAttachments     int   `json:"total_attachments"`
	DocumentAttachments  int   `json:"document_attachments"`
	SelectedAttachments  int   `json:"selected_attachments"`
	ExtractedAttachments int   `json:"extracted_attachments"`
	DownloadedBytes      int64 `json:"downloaded_bytes"`
	Limit                int   `json:"limit,omitempty"`
	MaxBytes             int64 `json:"max_bytes,omitempty"`
	MaxTotalBytes        int64 `json:"max_total_bytes,omitempty"`
	MaxChars             int   `json:"max_chars,omitempty"`
}

type ExtractedAttachmentText struct {
	Index           int    `json:"index"`
	ID              int    `json:"id,omitempty"`
	FileType        string `json:"file_type"`
	Name            string `json:"name,omitempty"`
	MIMEType        string `json:"mime_type,omitempty"`
	FileSize        int    `json:"file_size,omitempty"`
	DownloadedBytes int64  `json:"downloaded_bytes,omitempty"`
	Extractor       string `json:"extractor,omitempty"`
	Text            string `json:"text,omitempty"`
	TextChars       int    `json:"text_chars,omitempty"`
	Truncated       bool   `json:"truncated,omitempty"`
	SHA256          string `json:"sha256,omitempty"`
}

type attachmentDownload struct {
	Name     string
	MIMEType string
	Path     string
	Bytes    int64
	SHA256   string
	Cleanup  func()
	FinalURL string
}

var runPDFToText = func(ctx context.Context, path string) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", fmt.Errorf("pdftotext is required to extract PDF attachments: %w", err)
	}

	cmd := exec.CommandContext(ctx, "pdftotext", "-q", "-nopgbrk", path, "-")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			msg := strings.TrimSpace(string(exitErr.Stderr))
			if msg == "" {
				msg = exitErr.Error()
			}
			return "", fmt.Errorf("pdftotext failed: %s", msg)
		}
		return "", err
	}
	return string(out), nil
}

func (c *Client) ExtractConversationAttachments(ctx context.Context, conversationID int, opts ConversationAttachmentExtractOptions) (*ConversationAttachmentExtractResult, error) {
	opts = normalizeConversationAttachmentExtractOptions(opts)

	attachments, err := c.Conversations().Attachments(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	candidates, documentCount, err := selectAttachmentCandidates(attachments, opts)
	if err != nil {
		return nil, err
	}

	result := &ConversationAttachmentExtractResult{
		ConversationID: conversationID,
		Items:          make([]ExtractedAttachmentText, 0, len(candidates)),
		Meta: ConversationAttachmentMeta{
			TotalAttachments:    len(attachments),
			DocumentAttachments: documentCount,
			SelectedAttachments: len(candidates),
			Limit:               opts.Limit,
			MaxBytes:            opts.MaxBytes,
			MaxTotalBytes:       opts.MaxTotalBytes,
			MaxChars:            opts.MaxChars,
		},
	}

	downloadedTotal := int64(0)
	for _, candidate := range candidates {
		remainingTotal := opts.MaxTotalBytes
		if remainingTotal > 0 {
			remainingTotal -= downloadedTotal
			if remainingTotal <= 0 {
				return nil, fmt.Errorf("document downloads exceed total limit of %d bytes", opts.MaxTotalBytes)
			}
		}

		downloadLimit := opts.MaxBytes
		if downloadLimit > 0 && remainingTotal > 0 && remainingTotal < downloadLimit {
			downloadLimit = remainingTotal
		} else if downloadLimit == 0 {
			downloadLimit = remainingTotal
		}

		download, err := c.downloadAttachmentToTemp(ctx, candidate.Attachment.DataURL, downloadLimit)
		if err != nil {
			return nil, fmt.Errorf("attachment %d: %w", candidate.Index, err)
		}

		text, extractor, err := extractAttachmentText(ctx, download.Path, download.Name, download.MIMEType)
		if download.Cleanup != nil {
			download.Cleanup()
		}
		if err != nil {
			return nil, fmt.Errorf("attachment %d: %w", candidate.Index, err)
		}

		text, truncated := truncateText(text, opts.MaxChars)
		item := ExtractedAttachmentText{
			Index:           candidate.Index,
			ID:              candidate.Attachment.ID,
			FileType:        candidate.Attachment.FileType,
			Name:            download.Name,
			MIMEType:        download.MIMEType,
			FileSize:        candidate.Attachment.FileSize,
			DownloadedBytes: download.Bytes,
			Extractor:       extractor,
			Text:            text,
			TextChars:       len([]rune(text)),
			Truncated:       truncated,
			SHA256:          download.SHA256,
		}
		result.Items = append(result.Items, item)

		downloadedTotal += download.Bytes
	}

	result.Meta.ExtractedAttachments = len(result.Items)
	result.Meta.DownloadedBytes = downloadedTotal

	if result.Items == nil {
		result.Items = []ExtractedAttachmentText{}
	}
	return result, nil
}

type attachmentCandidate struct {
	Index      int
	Attachment Attachment
}

func normalizeConversationAttachmentExtractOptions(opts ConversationAttachmentExtractOptions) ConversationAttachmentExtractOptions {
	if opts.Limit < 0 {
		opts.Limit = 0
	}
	if opts.MaxChars < 0 {
		opts.MaxChars = 0
	}
	if opts.UnsafeNoSizeLimit {
		opts.MaxBytes = 0
		opts.MaxTotalBytes = 0
	}
	return opts
}

func selectAttachmentCandidates(attachments []Attachment, opts ConversationAttachmentExtractOptions) ([]attachmentCandidate, int, error) {
	candidates := make([]attachmentCandidate, 0, len(attachments))
	documentCount := 0
	indexMap := make(map[int]Attachment, len(attachments))
	for idx, att := range attachments {
		index := idx + 1
		indexMap[index] = att
		if IsDocumentAttachment(att) {
			documentCount++
		}
	}

	if len(opts.Indexes) > 0 {
		seen := make(map[int]struct{}, len(opts.Indexes))
		for _, index := range opts.Indexes {
			if index < 1 || index > len(attachments) {
				return nil, documentCount, fmt.Errorf("attachment index %d is out of range", index)
			}
			if _, ok := seen[index]; ok {
				continue
			}
			seen[index] = struct{}{}

			att := indexMap[index]
			if !IsDocumentAttachment(att) {
				return nil, documentCount, fmt.Errorf("attachment index %d is not a document attachment", index)
			}
			candidates = append(candidates, attachmentCandidate{
				Index:      index,
				Attachment: att,
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Index < candidates[j].Index
		})
		return candidates, documentCount, nil
	}

	for idx, att := range attachments {
		if !IsDocumentAttachment(att) {
			continue
		}
		candidates = append(candidates, attachmentCandidate{
			Index:      idx + 1,
			Attachment: att,
		})
		if opts.Limit > 0 && len(candidates) >= opts.Limit {
			break
		}
	}

	return candidates, documentCount, nil
}

func IsDocumentAttachment(att Attachment) bool {
	switch strings.ToLower(strings.TrimSpace(att.FileType)) {
	case "file", "document":
		return true
	default:
		return false
	}
}

func AttachmentDisplayName(att Attachment) string {
	return displayNameFromURL(att.DataURL)
}

func (c *Client) downloadAttachmentToTemp(ctx context.Context, rawURL string, maxBytes int64) (*attachmentDownload, error) {
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(rawURL); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	if maxBytes > 0 && resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("attachment too large: %d bytes exceeds %d", resp.ContentLength, maxBytes)
	}

	tmp, err := os.CreateTemp("", "chatwoot-attachment-*")
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	defer func() {
		_ = tmp.Close()
	}()

	limitReader := io.Reader(resp.Body)
	if maxBytes > 0 {
		limitReader = io.LimitReader(resp.Body, maxBytes+1)
	}

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), limitReader)
	if err != nil {
		cleanup()
		return nil, err
	}
	if maxBytes > 0 && written > maxBytes {
		cleanup()
		return nil, fmt.Errorf("attachment too large: exceeds %d bytes", maxBytes)
	}

	if err := tmp.Sync(); err != nil {
		cleanup()
		return nil, err
	}

	if _, err := tmp.Seek(0, 0); err != nil {
		cleanup()
		return nil, err
	}

	head := make([]byte, 512)
	n, _ := io.ReadFull(tmp, head)
	head = head[:n]

	name := attachmentFilename(resp, rawURL)
	mimeType := attachmentMIMEType(resp.Header.Get("Content-Type"), name, head)

	return &attachmentDownload{
		Name:     name,
		MIMEType: mimeType,
		Path:     tmp.Name(),
		Bytes:    written,
		SHA256:   hex.EncodeToString(hash.Sum(nil)),
		Cleanup:  cleanup,
		FinalURL: resp.Request.URL.String(),
	}, nil
}

func attachmentFilename(resp *http.Response, rawURL string) string {
	if resp != nil {
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			if _, params, err := mime.ParseMediaType(cd); err == nil {
				if filename := strings.TrimSpace(params["filename"]); filename != "" {
					return filename
				}
			}
		}
		if resp.Request != nil && resp.Request.URL != nil {
			if name := displayNameFromURL(resp.Request.URL.String()); name != "" {
				return name
			}
		}
	}
	return displayNameFromURL(rawURL)
}

func displayNameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

func attachmentMIMEType(contentType, name string, head []byte) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	switch {
	case contentType != "" && contentType != "application/octet-stream":
		return contentType
	case len(head) > 0:
		sniffed := http.DetectContentType(head)
		if sniffed != "" && sniffed != "application/octet-stream" {
			return sniffed
		}
	}

	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "application/pdf"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

func extractAttachmentText(ctx context.Context, filePath, name, mimeType string) (string, string, error) {
	switch normalizedDocumentType(name, mimeType) {
	case "pdf":
		text, err := runPDFToText(ctx, filePath)
		if err != nil {
			return "", "", err
		}
		return normalizeExtractedText(text), "pdftotext", nil
	case "docx":
		text, err := extractDOCXText(filePath)
		if err != nil {
			return "", "", err
		}
		return normalizeExtractedText(text), "docx-xml", nil
	case "xlsx":
		text, err := extractXLSXText(filePath)
		if err != nil {
			return "", "", err
		}
		return normalizeExtractedText(text), "xlsx-xml", nil
	case "text":
		text, err := extractPlainText(filePath)
		if err != nil {
			return "", "", err
		}
		return normalizeExtractedText(text), "text", nil
	default:
		return "", "", fmt.Errorf("unsupported document type %q", mimeType)
	}
}

func normalizedDocumentType(name, mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "application/pdf":
		return "pdf"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "text/plain", "text/markdown", "application/json", "text/csv", "application/xml", "text/xml", "text/html":
		return "text"
	}

	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	case ".txt", ".md", ".markdown", ".json", ".csv", ".xml", ".html", ".htm":
		return "text"
	default:
		return ""
	}
}

func extractPlainText(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(bytes.ToValidUTF8(data, []byte(" "))), nil
}

func extractDOCXText(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = reader.Close() }()

	parts := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "word/") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}
		base := path.Base(file.Name)
		if base != "document.xml" && !strings.HasPrefix(base, "header") && !strings.HasPrefix(base, "footer") && !strings.HasPrefix(base, "footnotes") && !strings.HasPrefix(base, "endnotes") {
			continue
		}
		parts = append(parts, file.Name)
	}
	sort.Strings(parts)

	if len(parts) == 0 {
		return "", fmt.Errorf("docx missing word/document.xml")
	}

	var sections []string
	for _, name := range parts {
		file, err := reader.Open(name)
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			return "", err
		}

		text, err := extractTextFromXML(data)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) != "" {
			sections = append(sections, text)
		}
	}

	return strings.Join(sections, "\n\n"), nil
}

func extractXLSXText(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = reader.Close() }()

	files := make(map[string]*zip.File, len(reader.File))
	sheets := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			sheets = append(sheets, file.Name)
		}
	}
	sort.Strings(sheets)
	if len(sheets) == 0 {
		return "", fmt.Errorf("xlsx missing worksheet data")
	}

	sharedStrings, err := readXLSXSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return "", err
	}

	sections := make([]string, 0, len(sheets))
	for _, sheetName := range sheets {
		file := files[sheetName]
		if file == nil {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", err
		}

		text, err := extractWorksheetText(data, sharedStrings)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) == "" {
			continue
		}

		sections = append(sections, fmt.Sprintf("[%s]\n%s", path.Base(sheetName), text))
	}

	return strings.Join(sections, "\n\n"), nil
}

func extractTextFromXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var builder strings.Builder
	needSpace := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}

		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "p", "br", "tab":
				if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "\n") {
					builder.WriteString("\n")
				}
				needSpace = false
			}
		case xml.CharData:
			text := strings.TrimSpace(string(tok))
			if text == "" {
				continue
			}
			if needSpace {
				builder.WriteByte(' ')
			}
			builder.WriteString(text)
			needSpace = true
		case xml.EndElement:
			if tok.Name.Local == "p" && builder.Len() > 0 && !strings.HasSuffix(builder.String(), "\n") {
				builder.WriteString("\n")
			}
		}
	}

	return builder.String(), nil
}

func normalizeExtractedText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\f", "\n\n")
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	text = strings.Join(lines, "\n")
	text = strings.TrimSpace(text)
	return text
}

func truncateText(text string, maxChars int) (string, bool) {
	if maxChars <= 0 {
		return text, false
	}
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text, false
	}
	return strings.TrimSpace(string(runes[:maxChars])), true
}

func readXLSXSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}

	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	decoder := xml.NewDecoder(rc)
	stringsOut := []string{}
	var builder strings.Builder
	collectText := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "si":
				builder.Reset()
			case "t":
				collectText = true
			}
		case xml.CharData:
			if collectText {
				builder.Write(tok)
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "t":
				collectText = false
			case "si":
				stringsOut = append(stringsOut, builder.String())
			}
		}
	}

	return stringsOut, nil
}

func extractWorksheetText(data []byte, sharedStrings []string) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rows := []string{}
	currentRow := []string{}
	currentCellType := ""
	currentCellValue := ""
	capturing := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}

		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "row":
				currentRow = currentRow[:0]
			case "c":
				currentCellType = ""
				currentCellValue = ""
				for _, attr := range tok.Attr {
					if attr.Name.Local == "t" {
						currentCellType = attr.Value
						break
					}
				}
			case "v", "t":
				capturing = true
			}
		case xml.CharData:
			if capturing {
				currentCellValue += string(tok)
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "v", "t":
				capturing = false
			case "c":
				value := strings.TrimSpace(currentCellValue)
				if currentCellType == "s" {
					if idx, err := strconv.Atoi(value); err == nil && idx >= 0 && idx < len(sharedStrings) {
						value = sharedStrings[idx]
					}
				}
				currentRow = append(currentRow, strings.TrimSpace(value))
			case "row":
				if len(currentRow) == 0 {
					continue
				}
				rows = append(rows, strings.TrimRight(strings.Join(currentRow, "\t"), "\t"))
			}
		}
	}

	return strings.Join(rows, "\n"), nil
}
