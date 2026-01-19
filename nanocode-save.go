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

// --- CONFIGURATION ---
const CurrentModel = "codestral-latest"
const MistralURL = "https://api.mistral.ai/v1/chat/completions"

var MistralKey = os.Getenv("MISTRAL_API_KEY")

// --- COULEURS ---
const (
	Reset  = "\033[0m"
	Cyan   = "\033[36m"
	Dim    = "\033[2m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Blue   = "\033[34m" // Ajouté
	Yellow = "\033[33m" // Ajouté
)

// --- STRUCTS (STREAMING COMPATIBLE) ---

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type RequestBody struct {
	Model       string        `json:"model"`
	Messages    []Message     `json:"messages"`
	Tools       []interface{} `json:"tools,omitempty"`
	ToolChoice  string        `json:"tool_choice,omitempty"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

// --- TOOLS ---

func toolRead(args map[string]interface{}) string {
	path, ok := args["path"].(string)
	if !ok { return "Error: path missing" }
	data, err := os.ReadFile(path)
	if err != nil { return "Error: " + err.Error() }
	if len(data) > 6000 { return string(data[:6000]) + "\n...[TRUNCATED]..." }
	return string(data)
}

func toolWrite(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil { return "Error: " + err.Error() }
	return "Success."
}

func toolBash(args map[string]interface{}) string {
	cmdStr, _ := args["cmd"].(string)
	cmd := exec.Command("bash", "-c", cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out; cmd.Stderr = &out
	err := cmd.Run()
	output := strings.TrimSpace(out.String())
	if err != nil { return fmt.Sprintf("Failed: %s\n%s", err.Error(), output) }
	if output == "" { return "Done (no output)" }
	return output
}

func toolGlob(args map[string]interface{}) string {
	pat, _ := args["pat"].(string)
	matches, _ := filepath.Glob(pat)
	if len(matches) == 0 { return "No matches" }
	sort.Strings(matches)
	return strings.Join(matches, "\n")
}

// --- LOGIC ---

func getTools() []interface{} {
	return []interface{}{
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "read", "description": "Read file", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]string{"type": "string"}}, "required": []string{"path"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "write", "description": "Write file", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]string{"type": "string"}, "content": map[string]string{"type": "string"}}, "required": []string{"path", "content"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "bash", "description": "Run shell cmd", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"cmd": map[string]string{"type": "string"}}, "required": []string{"cmd"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "glob", "description": "List files *", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"pat": map[string]string{"type": "string"}}, "required": []string{"pat"}}}},
	}
}

func callMistralStream(messages []Message) (string, []ToolCall, error) {
	reqBody := RequestBody{
		Model: CurrentModel, Messages: messages, Tools: getTools(), ToolChoice: "auto", Temperature: 0.1, Stream: true,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", MistralURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+MistralKey)

	// Utilisation de 'time' ici pour éviter l'erreur d'import
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", nil, fmt.Errorf("API Error %s", resp.Status)
	}

	reader := bufio.NewReader(resp.Body)
	fullContent := ""
	var toolCalls []ToolCall
	
	currentToolID := ""
	currentToolName := ""
	currentToolArgs := ""

	fmt.Printf("%s⏺ %s", Cyan, Reset)

	for {
		line, err := reader.ReadString('\n')
		if err != nil { break }
		
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") { continue }
		if line == "data: [DONE]" { break }
		
		jsonPart := line[6:]
		var chunk StreamResponse
		json.Unmarshal([]byte(jsonPart), &chunk)
		
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			
			if delta.Content != "" {
				fmt.Print(delta.Content)
				fullContent += delta.Content
			}

			if len(delta.ToolCalls) > 0 {
				tc := delta.ToolCalls[0]
				if tc.ID != "" {
					if currentToolID != "" {
						toolCalls = append(toolCalls, ToolCall{ID: currentToolID, Type: "function", Function: struct{Name string "json:\"name\""; Arguments string "json:\"arguments\""}{Name: currentToolName, Arguments: currentToolArgs}})
					}
					currentToolID = tc.ID
					currentToolName = tc.Function.Name
					currentToolArgs = ""
				}
				currentToolArgs += tc.Function.Arguments
			}
		}
	}
	fmt.Println()

	if currentToolID != "" {
		toolCalls = append(toolCalls, ToolCall{ID: currentToolID, Type: "function", Function: struct{Name string "json:\"name\""; Arguments string "json:\"arguments\""}{Name: currentToolName, Arguments: currentToolArgs}})
	}

	return fullContent, toolCalls, nil
}

// --- MAIN ---

func main() {
	if MistralKey == "" { fmt.Printf("%sErreur: MISTRAL_API_KEY manquante.%s\n", Red, Reset); return }
	
	cwd, _ := os.Getwd()
	fmt.Printf("%snanocode-stream%s | %s%s%s\n\n", Bold, Reset, Dim, CurrentModel, Reset)

	sysPrompt := "You are a coding assistant. CWD: " + cwd + ". " +
		"Use tools to inspect code. AFTER using a tool, you MUST summarize what you found."
	
	history := []Message{{Role: "system", Content: sysPrompt}}
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// 'Blue' est maintenant défini, plus d'erreur !
		fmt.Printf("%s%s❯%s ", Bold, Blue, Reset)
		if !scanner.Scan() { break }
		input := scanner.Text()
		if input == "/q" { break }
		if input == "/c" { history = []Message{{Role: "system", Content: sysPrompt}}; fmt.Printf("%sCleaned.%s\n", Green, Reset); continue }

		history = append(history, Message{Role: "user", Content: input})

		for {
			content, tools, err := callMistralStream(history)
			if err != nil { fmt.Printf("%sError: %v%s\n", Red, err, Reset); break }

			history = append(history, Message{Role: "assistant", Content: content, ToolCalls: tools})

			if len(tools) > 0 {
				for _, tool := range tools {
					fname := tool.Function.Name
					fmt.Printf("%s[EXEC: %s]%s\n", Green, strings.ToUpper(fname), Reset)
					
					var args map[string]interface{}
					json.Unmarshal([]byte(tool.Function.Arguments), &args)
					
					var res string
					switch fname {
					case "read": res = toolRead(args)
					case "write": res = toolWrite(args)
					case "bash": res = toolBash(args)
					case "glob": res = toolGlob(args)
					default: res = "unknown tool"
					}
					
					preview := strings.ReplaceAll(res, "\n", " ")
					if len(preview) > 60 { preview = preview[:60] + "..." }
					fmt.Printf("%s⎿ %s%s\n", Dim, preview, Reset)
					
					history = append(history, Message{Role: "tool", ToolCallID: tool.ID, Name: fname, Content: res})
				}
				continue
			}
			break
		}
		fmt.Println()
	}
}
