# Pax Chroot

Bootstraps a chroot for pax/paxOS development

## Usage

Pax Chroot requires root for setup, however it uses pax resources from the home folder. **If pax is installed in `/root/.apkg` on your system simply run the following command from within this folder:**

```bash
sudo go run .
```

Otherwise run the following command as the user with pax in their home folder **(this will be the case for most people)**:

```bash
sudo -E go run .
```