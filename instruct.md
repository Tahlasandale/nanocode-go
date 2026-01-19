# INSPIRATION ET CREDITS
https://github.com/1rgs/nanocode/tree/master

Here is the step-by-step guide to compiling and installing `nanocode` as a global command for macOS, Linux, and Windows.

**Prerequisite:** Open your terminal/console in the folder where your `nanocode.go` file is located.

```markdown
# How to Install 'nanocode' as a System Command

## 🍎 macOS

1. **Build the executable**
   ```bash
   go build -o nanocode nanocode.go

```

2. **Move it to the system folder** (requires password)
```bash
sudo mv nanocode /usr/local/bin/

```


3. **Set the API Key permanently** (Zsh is the default on modern Macs)
```bash
echo 'export GEMINI_API_KEY="YOUR_API_KEY_HERE"' >> ~/.zshrc
source ~/.zshrc

```


4. **Verify**
Open a new terminal window and type `nanocode`.

---

## 🐧 Linux (Ubuntu/Debian/Arch)

1. **Build the executable**
```bash
go build -o nanocode nanocode.go

```


2. **Move it to the system folder**
```bash
sudo mv nanocode /usr/local/bin/

```


3. **Set the API Key permanently** (Bash is usually the default)
```bash
echo 'export GEMINI_API_KEY="YOUR_API_KEY_HERE"' >> ~/.bashrc
source ~/.bashrc

```


4. **Verify**
Type `nanocode` in any terminal.

---

## 🪟 Windows (PowerShell)

*Run PowerShell as Administrator for these steps.*

1. **Build the executable**
```powershell
go build -o nanocode.exe nanocode.go

```


2. **Create a tools folder and move the file**
```powershell
mkdir C:\Tools
move nanocode.exe C:\Tools\

```


3. **Add that folder to your System PATH**
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Tools", "User")

```


*(Note: You might need to restart your computer or log out/in for the PATH to update completely).*
4. **Set the API Key permanently**
```powershell
setx GEMINI_API_KEY "YOUR_API_KEY_HERE"

```


5. **Verify**
Open a **new** PowerShell window and type `nanocode`.

```

### ⚠️ Important Note
Replace `"YOUR_API_KEY_HERE"` with your **new** valid API key. Do not use the one you posted in the chat previously, as it is compromised.

```