- Extract the compressed file to get the binary file
- Stop the current node
- Replace the current binary with the new binary file
- Verify the new binary file
  $oraid version
    0.50.12-tachyon-fix
  $oraid q wasm libwasmvm-version
  2.1.3
- Restart the node with the new binary file

This binary file runs on:
Architecture: x86_64
OS: Ubuntu 24.04 LTS
