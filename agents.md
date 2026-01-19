# Coding Guidelines
1. **Code Organization**: Maintain clear separation of concerns with distinct functions for each tool (e.g., `toolRead`, `toolWrite`).
2. **Error Handling**: Implement robust error handling for file operations and API calls.
3. **Environment Variables**: Use environment variables for sensitive data like API keys.
4. **ANSI Colors**: Utilize ANSI escape codes for terminal output formatting.
5. **Streaming Responses**: Support streaming responses for real-time interaction.
6. **Smart Commits**: Use conventional commit messages for clear version control history. Use git in the terminal to create commits.
7. **README Updates**: Update the README.md file when modifying project features. You gotta update the md code to match the new features.
8. **Languages**: When you write in the files do it al in english BUT when you talk to the user use french.

# Patterns

1. **Functional Tools**: Implement tools as functions with consistent interfaces (e.g., `toolRead`, `toolWrite`).
2. **Structured Data**: Use Go structs to model API requests and responses.
3. **Command Execution**: Execute shell commands using `exec.Command` with proper output handling.
4. **File Operations**: Perform file operations with `os.ReadFile` and `os.WriteFile`.
5. **Stream Processing**: Process streaming responses from the API for real-time updates.

# Architecture Notes

1. **Modular Design**: The application is designed with modular components for tools, API interactions, and command execution.
2. **API Integration**: Integrate with the Mistral API for natural language processing and command interpretation.
3. **Tool Abstraction**: Abstract tool functionality to allow easy addition or modification of tools.
4. **User Interaction**: Provide a conversational interface for user interaction and command execution.
5. **Cross-Platform Support**: Support installation and usage across macOS, Linux, and Windows platforms.

### PDF HANDLING PROTOCOL
The agent cannot read binary .pdf files directly.
If the user asks to analyze or read a PDF file (e.g., `doc.pdf`), you MUST follow this strictly:

1. **CONVERT**: Use the `bash` tool to run: `pdftomd doc.pdf doc.md`
2. **ReWrite**: Use the `write` tool to format correctly the md file with titles and structure and fix errors or broken strings. (IMPORTANT) to make it cool to read for the user.
3. **READ**: Use the `read` tool to read the newly created `doc.md` file.
4. **ANALYZE**: Summarize or analyze the content of the markdown file.
5. **CLEANUP** (Optional): You may remove the .md file afterwards if requested.
