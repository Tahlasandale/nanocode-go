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
	Reset   = "\033[0m"
	Cyan    = "\033[36m"
	Dim     = "\033[2m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Blue    = "\033[34m"
	Yellow  = "\033[33m"
	Magenta = "\033[35m"
)

// --- STRUCTS ---

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

// --- OUTILS ---

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

func getTools() []interface{} {
	return []interface{}{
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "read", "description": "Read file", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]string{"type": "string"}}, "required": []string{"path"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "write", "description": "Write file", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]string{"type": "string"}, "content": map[string]string{"type": "string"}}, "required": []string{"path", "content"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "bash", "description": "Run shell cmd", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"cmd": map[string]string{"type": "string"}}, "required": []string{"cmd"}}}},
		map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "glob", "description": "List files *", "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"pat": map[string]string{"type": "string"}}, "required": []string{"pat"}}}},
	}
}

// --- MOTEUR IA (STREAMING) ---

func callMistralStream(messages []Message) (string, []ToolCall, error) {
	reqBody := RequestBody{
		Model: CurrentModel, Messages: messages, Tools: getTools(), ToolChoice: "auto", Temperature: 0.1, Stream: true,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", MistralURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+MistralKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return "", nil, fmt.Errorf("API Error %s", resp.Status) }

	reader := bufio.NewReader(resp.Body)
	fullContent := ""
	var toolCalls []ToolCall
	
	currentToolID, currentToolName, currentToolArgs := "", "", ""
	fmt.Printf("%s", Magenta) // Pensée en violet

	for {
		line, err := reader.ReadString('\n')
		if err != nil { break }
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") { continue }
		if line == "data: [DONE]" { break }
		
		var chunk StreamResponse
		json.Unmarshal([]byte(line[6:]), &chunk)
		
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
					currentToolID = tc.ID; currentToolName = tc.Function.Name; currentToolArgs = ""
				}
				currentToolArgs += tc.Function.Arguments
			}
		}
	}
	fmt.Printf("%s\n", Reset)

	if currentToolID != "" {
		toolCalls = append(toolCalls, ToolCall{ID: currentToolID, Type: "function", Function: struct{Name string "json:\"name\""; Arguments string "json:\"arguments\""}{Name: currentToolName, Arguments: currentToolArgs}})
	}
	return fullContent, toolCalls, nil
}

// --- GESTION CONTEXTE & ANALYSE ---

func getSystemPrompt(cwd string) string {
	// 1. Base Prompt (Orchestrator Rules)
	p := "You are the Orchestrator Agent. CWD: " + cwd + ".\n" +
		"PROTOCOL: THOUGHT (Explain plan) > ACTION (Use tool) > OBSERVATION > REPEAT.\n" +
		"Never use a tool without explaining WHY first in the THOUGHT phase."
	
	// 2. agents.md (Mémoire persistante)
	if data, err := os.ReadFile("agents.md"); err == nil {
		p += "\n\n=== [agents.md] MEMORY & GUIDELINES ===\n" + string(data)
	}
	return p
}

func analyzeProject() string {
	files, _ := filepath.Glob("*")
	var contentBuilder strings.Builder
	contentBuilder.WriteString("Analyze these project files. Output a clean Markdown list of Coding Guidelines, patterns, and Architecture notes (max 300 words). Do NOT act as an agent, just output the MD content:\n")

	count := 0
	for _, f := range files {
		// Exclusion des fichiers binaires, cachés ou mémoire
		if strings.HasPrefix(f, ".") || f == "nanocode" || f == "agents.md" { continue }
		info, err := os.Stat(f)
		if err == nil && !info.IsDir() {
			data, _ := os.ReadFile(f)
			if len(data) > 3000 { data = data[:3000] } // Tronque les gros fichiers
			contentBuilder.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", f, string(data)))
			count++
		}
	}
	
	if count == 0 { return "No files to analyze." }

	msgs := []Message{{Role: "user", Content: contentBuilder.String()}}
	fmt.Printf("%s(Analyzing project structure to update agents.md...)%s\n", Yellow, Reset)
	
	// On utilise la fonction de stream pour voir l'analyse en direct, et on récupère le texte
	resp, _, err := callMistralStream(msgs)
	if err != nil { return "" }
	return resp
}

// --- MAIN ---

func main() {
	if MistralKey == "" { fmt.Printf("%sErreur: MISTRAL_API_KEY manquante.%s\n", Red, Reset); return }
	
	cwd, _ := os.Getwd()
	sysPrompt := getSystemPrompt(cwd)
	
	fmt.Printf("%snanocode-v7 (Persistent Memory)%s | %s%s%s\n", Bold, Reset, Dim, CurrentModel, Reset)
	fmt.Printf("Commands: %s/i%s (Init/Update Memory), %s/c%s (Clear Chat), %s/q%s (Quit)\n\n", Green, Reset, Green, Reset, Green, Reset)

	history := []Message{{Role: "system", Content: sysPrompt}}
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("%s%s❯%s ", Bold, Blue, Reset)
		if !scanner.Scan() { break }
		input := scanner.Text()

		if input == "/q" { break }
		
		// --- COMMANDE /i : ANALYSE ET SAUVEGARDE ---
		if input == "/i" {
			guidelines := analyzeProject()
			if guidelines != "" {
				header := fmt.Sprintf("\n\n### AUTO-ANALYSIS (%s) ###\n", time.Now().Format("2006-01-02 15:04"))
				
				// Ajout à agents.md
				f, err := os.OpenFile("agents.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					f.WriteString(header + guidelines)
					f.Close()
					fmt.Printf("%s[agents.md updated on disk]%s\n", Green, Reset)
				} else {
					fmt.Printf("%sError writing agents.md: %v%s\n", Red, err, Reset)
				}

				// Rechargement immédiat du cerveau
				sysPrompt = getSystemPrompt(cwd)
				history = []Message{{Role: "system", Content: sysPrompt}}
				fmt.Printf("%sContext reloaded from agents.md.%s\n", Green, Reset)
			}
			continue
		}

		if input == "/c" {
			sysPrompt = getSystemPrompt(cwd) // Relecture fraiche du fichier
			history = []Message{{Role: "system", Content: sysPrompt}}
			fmt.Printf("%sCleaned & Memory Reloaded.%s\n", Green, Reset)
			continue
		}

		history = append(history, Message{Role: "user", Content: input})

		// --- BOUCLE ORCHESTRATEUR ---
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
				fmt.Printf("%s(🔄 Orchestrator analyzing result...)%s\n", Yellow, Reset)
				continue
			}
			break
		}
		fmt.Println()
	}
}
