#!/bin/bash

# --- nanocode-go Linux Installer Script ---

# Configuration
EXEC_NAME="nanocode"
INSTALL_PATH="/usr/local/bin/"

# --- 1. Ask for API Key ---
echo "--- nanocode-go Linux Installer ---"
read -p "Enter your Mistral API Key: " API_KEY

if [ -z "$API_KEY" ]; then
    echo "Error: API Key cannot be empty. Exiting."
    exit 1
fi

# --- 2. Check for Go installation ---
if ! command -v go &> /dev/null
then
    echo "Go is not installed. Installing Go..."
    # Install Go using package manager
    if command -v apt-get &> /dev/null
    then
        sudo apt-get update
        sudo apt-get install -y golang
    elif command -v dnf &> /dev/null
    then
        sudo dnf install -y golang
    elif command -v yum &> /dev/null
    then
        sudo yum install -y golang
    else
        echo "Error: Unsupported package manager. Please install Go manually."
        exit 1
    fi
    echo "Go has been installed."
fi

echo "Go installation found."

# --- 3. Build the project ---
echo "Building nanocode executable..."
go build -o $EXEC_NAME nanocode.go
if [ $? -ne 0 ]; then
    echo "Error: Go build failed. Check nanocode.go for errors."
    exit 1
fi
echo "Build successful."

# --- 4. Move the executable to PATH ---
echo "Moving $EXEC_NAME to $INSTALL_PATH (requires sudo)..."
sudo mv $EXEC_NAME $INSTALL_PATH
if [ $? -ne 0 ]; then
    echo "Error: Failed to move executable. Check permissions or install path."
    exit 1
fi

# --- 5. Build and install pdftomd ---
echo "Building pdftomd executable..."
go build -o pdftomd pdftomd.go
if [ $? -ne 0 ]; then
    echo "Error: Go build failed. Check pdftomd.go for errors."
    exit 1
fi
echo "Build successful."

echo "Moving pdftomd to $INSTALL_PATH (requires sudo)..."
sudo mv pdftomd $INSTALL_PATH
if [ $? -ne 0 ]; then
    echo "Error: Failed to move executable. Check permissions or install path."
    exit 1
fi

# --- 6. Add API Key to shell profile ---
PROFILE_FILE=""
# Check for Bash (most common on Linux) then Zsh
if [ -f "$HOME/.bashrc" ]; then
    PROFILE_FILE="$HOME/.bashrc"
elif [ -f "$HOME/.zshrc" ]; then
    PROFILE_FILE="$HOME/.zshrc"
elif [ -f "$HOME/.profile" ]; then
    PROFILE_FILE="$HOME/.profile"
fi

if [ -n "$PROFILE_FILE" ]; then
    echo "Adding MISTRAL_API_KEY to $PROFILE_FILE..."
    # Check if the key is already set (using standard Linux sed)
    if grep -q "MISTRAL_API_KEY" "$PROFILE_FILE"; then
        sed -i "/MISTRAL_API_KEY/c\export MISTRAL_API_KEY=\"$API_KEY\"" "$PROFILE_FILE"
        echo "Updated existing MISTRAL_API_KEY."
    else
        echo -e "\n# Added by nanocode-go installer" >> "$PROFILE_FILE"
        echo "export MISTRAL_API_KEY=\"$API_KEY\"" >> "$PROFILE_FILE"
        echo "API Key added."
    fi

    # --- 7. Source the profile ---
echo "Sourcing $PROFILE_FILE to apply changes to the current session..."
source "$PROFILE_FILE" 2>/dev/null || true
fi

# --- 8. Verification ---
echo
"--- Installation Complete! ---"
echo "The 'nanocode' and 'pdftomd' commands are now installed."
echo "You can start a new terminal session or test it now by running: $EXEC_NAME"
echo