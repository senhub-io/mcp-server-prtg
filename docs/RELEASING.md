# Releasing

## Betas

A tag `X.Y.Z-beta.N` pushed on `dev` publishes a pre-release (workflow `dev-beta-release.yml`): Windows zip, Linux archives, MSI and `SHA256SUMS`. Betas are not signed.

## Production releases (signed)

The Authenticode certificate (Certum SimplySign) can only be used from the maintainer's signing Mac, so a production release is finished locally.

1. **Release PR**: merge `dev` into `main` through a pull request. `main-release.yml` derives the version from the latest beta tag (`2.0.0-beta.4` gives `2.0.0`), tags the merge commit and creates a **draft** release with the Windows zip and the Linux archives.
2. **Sign** on the signing Mac, with a SimplySign Desktop session open:
   ```sh
   scripts/sign-release.sh 2.0.0
   ```
   The script signs `mcp-server-prtg.exe` inside the Windows zip, builds the MSI from that signed executable (workflow `windows-msi.yml` with `exe_from_release=true`), signs the MSI, and refreshes the `.sha256` file and `SHA256SUMS`. It refuses a release that is not a draft.
3. **Check** on Windows: `Get-AuthenticodeSignature .\mcp-server-prtg.exe` and on the MSI must report `Status: Valid`.
4. **Publish**: `gh release edit 2.0.0 --draft=false`.

Prerequisites on the signing Mac: `gh` with write access to the repository, Java 17 (Temurin), the SimplySign PKCS#11 module, and optionally `pkcs11-tool` (OpenSC) and `osslsigncode` for the pre-checks and local verification performed by `packaging/windows/sign-authenticode.sh`.
