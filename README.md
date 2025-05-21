# HSMTool: Cross-Platform Key Management & Crypto Utility

HSMTool is a cross-platform desktop application for cryptographic professionals to securely manage cryptographic keys, interact with Hardware Security Modules (HSMs), and perform advanced cryptographic operations. Built in Go, it leverages the [Fyne](https://fyne.io) UI framework for a native look and feel on Windows, macOS, and Linux.

## Features

### 1. Key Manager
- **Create, import, and manage cryptographic keys** (ZMK, ZPK, TMK, PVK, KEK, AES, etc.).
- **Key generation via HSM**: Securely generate keys inside the HSM.
- **Check Value (KCV) calculation**: Auto-computed and validated.
- **Store key metadata**: Secure local storage (file or DB, not plaintext).
- **Table view**: List, filter, and manage stored keys with context menu (View, Export, Delete).
- **Validation indicators**: Color-coded status and check marks for key validity.

### 2. DES Calculator
- **Encrypt/Decrypt data** using DES/3DES with selectable modes (ECB, CBC, CFB).
- **Padding options**: ISO 9797-1 method 1/2, None.
- **KCV auto-calculation** for provided keys.
- **User-friendly UI**: Input validation, result display, and error feedback.

### 3. Bitwise Calculator
- **Bitwise operations**: XOR, AND, OR, NOT on hex data blocks.
- **Key Sharing Mode**: Split/Combine DES keys into components using XOR, with parity options.
- **Live KCV calculation** for all components.
- **Real-time validation**: Error highlights and tooltips for invalid input.

### 4. Settings
- **Configure HSM connection**: Set HSM IP and port.
- **Test connection**: Verify HSM availability from the UI.

### 5. HSM Command Sender
- **Send raw Thanles host commands** to the HSM.
- **Batch and timed requests**: Specify count or duration.
- **Response tracking**: Match responses by random header, real-time log with timestamps.
- **Progress indicators**: See status and completion in real time.

## Security & Best Practices
- **No plaintext key export**: Only encrypted export under ZMK/LMK.
- **Sensitive fields auto-clear** after use.
- **All crypto operations** routed through validated plugin/HSM interface.
- **No key data stored in memory longer than required.**

## Technology Stack
- **Language:** Go (>=1.18)
- **UI:** [Fyne](https://fyne.io) (cross-platform GUI)
- **Networking:** [anet](https://github.com/Andrei-cloud/anet) for robust HSM communication

## Building & Running

1. **Install Go** (https://golang.org/dl/)
2. **Clone the repository:**
   ```sh
   git clone https://github.com/andrei-cloud/hsmtool.git
   cd hsmtool
   ```
3. **Build the application:**
   ```sh
   make build
   # or
   go build -o hsmtool ./cmd/hsmtool
   ```
4. **Run:**
   ```sh
   ./hsmtool
   ```

## UI Overview
- **Tabbed interface**: Key Manager, DES Calculator, Bitwise Calculator, Settings, HSM Command Sender.
- **Consistent layout**: Action buttons, validation icons, and responsive design.
- **Light and dark themes** supported.


## License
MIT License. See [LICENSE](LICENSE).
