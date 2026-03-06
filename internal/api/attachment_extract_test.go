package api

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractConversationAttachments_TextDocument(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/attachments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload":[
				{"id": 1, "file_type": "image", "data_url": "https://example.com/image.png", "file_size": 20},
				{"id": 2, "file_type": "file", "data_url": "` + server.URL + `/files/note.txt", "file_size": 22}
			]}`))
		case "/files/note.txt":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("first line\nsecond line\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	result, err := client.ExtractConversationAttachments(context.Background(), 123, ConversationAttachmentExtractOptions{
		Limit:         DefaultDocumentExtractLimit,
		MaxBytes:      DefaultDocumentExtractMaxBytes,
		MaxTotalBytes: DefaultDocumentExtractMaxTotalBytes,
		MaxChars:      12,
	})
	if err != nil {
		t.Fatalf("ExtractConversationAttachments returned error: %v", err)
	}

	if result.Meta.TotalAttachments != 2 || result.Meta.DocumentAttachments != 1 {
		t.Fatalf("unexpected meta: %#v", result.Meta)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 extracted item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.Index != 2 {
		t.Fatalf("expected extracted item index 2, got %d", item.Index)
	}
	if item.Extractor != "text" {
		t.Fatalf("expected text extractor, got %q", item.Extractor)
	}
	if item.MIMEType != "text/plain" {
		t.Fatalf("expected text/plain mime, got %q", item.MIMEType)
	}
	if !item.Truncated {
		t.Fatalf("expected truncated text")
	}
	if item.Text != "first line\ns" {
		t.Fatalf("unexpected extracted text %q", item.Text)
	}
}

func TestExtractConversationAttachments_PDFUsesRunner(t *testing.T) {
	original := runPDFToText
	runPDFToText = func(ctx context.Context, path string) (string, error) {
		return "pdf extracted text", nil
	}
	defer func() {
		runPDFToText = original
	}()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/attachments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload":[
				{"id": 7, "file_type": "file", "data_url": "` + server.URL + `/files/sample.pdf", "file_size": 12}
			]}`))
		case "/files/sample.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	result, err := client.ExtractConversationAttachments(context.Background(), 123, ConversationAttachmentExtractOptions{
		Limit:         1,
		MaxBytes:      DefaultDocumentExtractMaxBytes,
		MaxTotalBytes: DefaultDocumentExtractMaxTotalBytes,
		MaxChars:      DefaultDocumentExtractMaxChars,
	})
	if err != nil {
		t.Fatalf("ExtractConversationAttachments returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Extractor != "pdftotext" {
		t.Fatalf("expected pdftotext extractor, got %q", result.Items[0].Extractor)
	}
	if result.Items[0].Text != "pdf extracted text" {
		t.Fatalf("unexpected pdf text %q", result.Items[0].Text)
	}
}

func TestExtractConversationAttachments_DOCX(t *testing.T) {
	docx := makeTestDOCX(t, "Hello from DOCX")

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/attachments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload":[
				{"id": 4, "file_type": "file", "data_url": "` + server.URL + `/files/sample.docx", "file_size": 12}
			]}`))
		case "/files/sample.docx":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
			_, _ = w.Write(docx)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	result, err := client.ExtractConversationAttachments(context.Background(), 123, ConversationAttachmentExtractOptions{
		Limit:         1,
		MaxBytes:      DefaultDocumentExtractMaxBytes,
		MaxTotalBytes: DefaultDocumentExtractMaxTotalBytes,
		MaxChars:      DefaultDocumentExtractMaxChars,
	})
	if err != nil {
		t.Fatalf("ExtractConversationAttachments returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Extractor != "docx-xml" {
		t.Fatalf("expected docx-xml extractor, got %q", result.Items[0].Extractor)
	}
	if !strings.Contains(result.Items[0].Text, "Hello from DOCX") {
		t.Fatalf("unexpected docx text %q", result.Items[0].Text)
	}
}

func TestExtractConversationAttachments_XLSX(t *testing.T) {
	xlsx := makeTestXLSX(t)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/attachments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload":[
				{"id": 5, "file_type": "file", "data_url": "` + server.URL + `/files/sample.xlsx", "file_size": 12}
			]}`))
		case "/files/sample.xlsx":
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			_, _ = w.Write(xlsx)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	result, err := client.ExtractConversationAttachments(context.Background(), 123, ConversationAttachmentExtractOptions{
		Limit:         1,
		MaxBytes:      DefaultDocumentExtractMaxBytes,
		MaxTotalBytes: DefaultDocumentExtractMaxTotalBytes,
		MaxChars:      DefaultDocumentExtractMaxChars,
	})
	if err != nil {
		t.Fatalf("ExtractConversationAttachments returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Extractor != "xlsx-xml" {
		t.Fatalf("expected xlsx-xml extractor, got %q", result.Items[0].Extractor)
	}
	if !strings.Contains(result.Items[0].Text, "Product\tPrice") || !strings.Contains(result.Items[0].Text, "Chocolate\t9.99") {
		t.Fatalf("unexpected xlsx text %q", result.Items[0].Text)
	}
}

func TestExtractConversationAttachments_InvalidIndex(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/attachments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload":[
				{"id": 4, "file_type": "file", "data_url": "` + server.URL + `/files/sample.txt", "file_size": 12}
			]}`))
		case "/files/sample.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("hi"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	_, err := client.ExtractConversationAttachments(context.Background(), 123, ConversationAttachmentExtractOptions{
		Indexes: []int{2},
		Limit:   1,
	})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out of range error, got %v", err)
	}
}

func makeTestDOCX(t *testing.T, text string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create("word/document.xml")
	if err != nil {
		t.Fatalf("Create(document.xml) failed: %v", err)
	}
	if _, err := writer.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`)); err != nil {
		t.Fatalf("Write(document.xml) failed: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close zip failed: %v", err)
	}
	return buf.Bytes()
}

func makeTestXLSX(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	sharedStrings, err := zipWriter.Create("xl/sharedStrings.xml")
	if err != nil {
		t.Fatalf("Create(sharedStrings.xml) failed: %v", err)
	}
	if _, err := sharedStrings.Write([]byte(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Product</t></si><si><t>Price</t></si><si><t>Chocolate</t></si></sst>`)); err != nil {
		t.Fatalf("Write(sharedStrings.xml) failed: %v", err)
	}

	sheet, err := zipWriter.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("Create(sheet1.xml) failed: %v", err)
	}
	if _, err := sheet.Write([]byte(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><v>9.99</v></c></row></sheetData></worksheet>`)); err != nil {
		t.Fatalf("Write(sheet1.xml) failed: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close zip failed: %v", err)
	}
	return buf.Bytes()
}
