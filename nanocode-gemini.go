package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// --- Configuration ---
const CurrentModel = "gemini-2.5-flash-preview-09-2025"

var (
	GeminiKey = os.Getenv("GEMINI_API_KEY")
)

// --- ANSI Colors ---
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

// --- Structs ---
type Part struct {
	Text         string            `json:"text,omitempty"`
	FunctionCall *FunctionCall     `json:"functionCall,omitempty"`
	FuncResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type FunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type Tool struct {
	FunctionDeclarations []FunctionDecl `json:"functionDeclarations"`
}

type FunctionDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  ParamSchema `json:"parameters"`
}

type ParamSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type RequestBody struct {
	Contents          []Content `json:"contents"`
	Tools             []Tool    `json:"tools,omitempty"`
	SystemInstruction *Content  `json:"systemInstruction,omitempty"`
}

type ResponseBody struct {
	Candidates []struct {
		Content Content `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// --- Outils ---
func toolRead(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	data, err := os.ReadFile(path)
	if err != nil { return "error: " + err.Error() }
	lines := strings.Split(string(data), "\n")
	offset := 0
	if v, ok := args["offset"]; ok { offset = int(v.(float64)) }
	limit := len(lines)
	if v, ok := args["limit"]; ok { limit = int(v.(float64)) }
	end := offset + limit
	if end > len(lines) { end = len(lines) }
	if offset >= len(lines) { return "EOF" }
	var sb strings.Builder
	for i, line := range lines[offset:end] {
		sb.WriteString(fmt.Sprintf("%4d| %s\n", offset+i+1, line))
	}
	return sb.String()
}

func toolWrite(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil { return "error: " + err.Error() }
	return "ok"
}

func toolEdit(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	oldStr, _ := args["old"].(string)
	newStr, _ := args["new"].(string)
	data, err := os.ReadFile(path)
	if err != nil { return "error: " + err.Error() }
	text := string(data)
	if !strings.Contains(text, oldStr) { return "error: old_string not found" }
	count := strings.Count(text, oldStr)
	replaceAll := false
	if v, ok := args["all"]; ok { replaceAll = v.(bool) }
	if count > 1 && !replaceAll { return fmt.Sprintf("error: old_string appears %d times, use all=true", count) }
	n := 1
	if replaceAll { n = -1 }
	newText := strings.Replace(text, oldStr, newStr, n)
	if err := os.WriteFile(path, []byte(newText), 0644); err != nil { return "error: " + err.Error() }
	return "ok"
}

func toolGlob(args map[string]interface{}) string {
	pat, _ := args["pat"].(string)
	root := "."
	if v, ok := args["path"]; ok { root = v.(string) }
	pattern := filepath.Join(root, pat)
	matches, err := filepath.Glob(pattern)
	if err != nil { return "error: " + err.Error() }
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})
	if len(matches) == 0 { return "none" }
	return strings.Join(matches, "\n")
}

func toolBash(args map[string]interface{}) string {
	cmdStr, _ := args["cmd"].(string)
	cmd := exec.Command("bash", "-c", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case <-time.After(30 * time.Second):
		if cmd.Process != nil { cmd.Process.Kill() }
		return out.String() + "\n(timed out)"
	case err := <-done:
		if err != nil { return out.String() + "\n" + err.Error() }
	}
	res := strings.TrimSpace(out.String())
	if res == "" { return "(empty)" }
	return res
}

// --- Main Logic ---

func defineTools() []Tool {
	return []Tool{{
		FunctionDeclarations: []FunctionDecl{
			{Name: "read", Description: "Read file", Parameters: ParamSchema{Type: "OBJECT", Required: []string{"path"}, Properties: map[string]Property{"path": {Type: "STRING"}, "offset": {Type: "INTEGER"}, "limit": {Type: "INTEGER"}}}},
			{Name: "write", Description: "Write file", Parameters: ParamSchema{Type: "OBJECT", Required: []string{"path", "content"}, Properties: map[string]Property{"path": {Type: "STRING"}, "content": {Type: "STRING"}}}},
			{Name: "edit", Description: "Replace string", Parameters: ParamSchema{Type: "OBJECT", Required: []string{"path", "old", "new"}, Properties: map[string]Property{"path": {Type: "STRING"}, "old": {Type: "STRING"}, "new": {Type: "STRING"}, "all": {Type: "BOOLEAN"}}}},
			{Name: "bash", Description: "Run shell cmd", Parameters: ParamSchema{Type: "OBJECT", Required: []string{"cmd"}, Properties: map[string]Property{"cmd": {Type: "STRING"}}}},
			{Name: "glob", Description: "List files", Parameters: ParamSchema{Type: "OBJECT", Required: []string{"pat"}, Properties: map[string]Property{"pat": {Type: "STRING"}, "path": {Type: "STRING"}}}},
		},
	}}
}

func callGemini(history []Content, sysPrompt string) (*ResponseBody, error) {
	// URL Directe v1beta standard
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + CurrentModel + ":generateContent?key=" + GeminiKey

	body := RequestBody{
		Contents:          history,
		Tools:             defineTools(),
		SystemInstruction: &Content{Parts: []Part{{Text: sysPrompt}}},
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var geminiResp ResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		msg := resp.Status
		if geminiResp.Error != nil {
			msg = fmt.Sprintf("%s - %s", resp.Status, geminiResp.Error.Message)
		}
		return nil, fmt.Errorf("API Error: %s", msg)
	}

	return &geminiResp, nil
}

func main() {
	if GeminiKey == "" {
		fmt.Printf("%sError: GEMINI_API_KEY not set%s\n", Red, Reset)
		return
	}

	cwd, _ := os.Getwd()
	fmt.Printf("%snanocode-go%s | %s%s%s | %s\n\n", Bold, Reset, Dim, CurrentModel, Reset, cwd)

	scanner := bufio.NewScanner(os.Stdin)
	var history []Content
	sysPrompt := fmt.Sprintf("You are a concise coding assistant. CWD: %s", cwd)

	for {
		fmt.Printf("%s%s❯%s ", Bold, Blue, Reset)
		if !scanner.Scan() { break }
		input := scanner.Text()
		
		if input == "/q" || input == "exit" { break }
		if input == "/c" {
			history = []Content{}
			fmt.Printf("%s⏺ Cleared%s\n", Green, Reset)
			continue
		}

		history = append(history, Content{Role: "user", Parts: []Part{{Text: input}}})

		for {
			resp, err := callGemini(history, sysPrompt)
			if err != nil {
				fmt.Printf("%s%v%s\n", Red, err, Reset)
				break
			}

			if len(resp.Candidates) == 0 {
				fmt.Println("No response from model.")
				break
			}

			candidate := resp.Candidates[0].Content
			history = append(history, candidate)

			hasFunc := false
			var functionResponses []Part

			for _, part := range candidate.Parts {
				if part.Text != "" {
					fmt.Printf("\n%s⏺%s %s\n", Cyan, Reset, part.Text)
				}

				if part.FunctionCall != nil {
					hasFunc = true
					fc := part.FunctionCall
					fmt.Printf("\n%s⏺ %s%s\n", Green, strings.ToUpper(fc.Name), Reset)
					
					var result string
					switch fc.Name {
					case "read": result = toolRead(fc.Args)
					case "write": result = toolWrite(fc.Args)
					case "edit": result = toolEdit(fc.Args)
					case "glob": result = toolGlob(fc.Args)
					case "bash": result = toolBash(fc.Args)
					default: result = "unknown tool"
					}
					
					// Preview simple
					preview := strings.ReplaceAll(result, "\n", " ")
					if len(preview) > 60 { preview = preview[:60] + "..." }
					fmt.Printf("  %s⎿ %s%s\n", Dim, preview, Reset)

					functionResponses = append(functionResponses, Part{
						FuncResponse: &FunctionResponse{
							Name: fc.Name, 
							Response: map[string]interface{}{"result": result},
						},
					})
				}
			}

			if !hasFunc { break }
			history = append(history, Content{Role: "user", Parts: functionResponses})
		}
		fmt.Println()
	}
}