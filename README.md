# nanocode-go

`nanocode-go` is an interactive coding assistant built with Go, leveraging the Mistral API to understand and execute commands. It provides a conversational interface where you can ask the AI to perform file operations, run shell commands, and assist with coding tasks directly from your terminal.

Inspired by nanocode: https://github.com/1rgs/nanocode

## Project Information

*   **Lines of Code**: 320
*   **Initial Development**: Coded with Gemini
*   **Improvement**: Further developed and improved using nanocode-go

## Features

*   **Mistral API Integration**: Utilizes the 
`codestral-latest` model (as defined in 
`nanocode.go`) for intelligent command interpretation and response generation.
*   **Custom Toolset**:      
    *   `read`: Read the content of a specified file (with optional offset/limit).
    *   `write`: Write content to a file.
    *   `edit`: Replace occurrences of a string within a file (with optional `all=true` for global replacement).
    *   `glob`: List files matching a given pattern (with optional root path).
    *   `bash`: Execute arbitrary shell commands.
*   **Conversational Interface**: Interact with the AI naturally through a command-line interface.
*   **Session Management**: Supports clearing the conversation history (`/c`).
*   **Colorful Output**: Uses ANSI escape codes for enhanced readability in the terminal.

## Getting Started

### Prerequisites

*   Go (version 1.18 or higher) installed on your system.
*   A Mistral API key. You can obtain one from [Mistral AI](https://mistral.ai/).

### Installation and Usage (as a global command)

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/your-username/nanocode-go.git # Replace with actual repo URL
    cd nanocode-go
    ```

2.  **Build the executable**:
    ```bash
    go build -o nanocode nanocode.go
    ```

3.  **Install globally and set API Key**:

    **🍎 macOS / 🐧 Linux (Bash/Zsh)**
    ```bash
    # Use the provided installer script
    chmod +x install_macos.sh
    ./install_macos.sh
    ```
    
    **🪟 Windows (PowerShell - Run as Administrator)**
    ```powershell
    # Use the provided installer script
    Set-ExecutionPolicy Bypass -Scope Process -Force
    .\install.ps1
    ```

4.  **Run the application**:
    Open a new terminal and simply type 
`nanocode`.
    
    You will then see a prompt like 
`nanocode-go | codestral-latest | /current/working/directory` and a 
`❯` where you can type your commands or questions.

## Commands

*   Type your natural language query or command.
*   ``/q`` or ``exit``: Quit the application.
*   ``/c``: Clear the conversation history.

## Example Interaction

```
nanocode-go | codestral-latest | /Users/bison/Documents/nanocode-go

❯ list all .go files
⏺ GLOB
  ⎿ nanocode.go
⏺ nanocode.go
```
