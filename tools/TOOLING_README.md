# Architecture repo tools

This repo also contains tools to automate parts of the project.

## Requirements
 - Go
 - Chrome
 - Google Drive API access

## Google Drive API access Setup
 - log into the corporate Google account
 - have or create a Google Cloud project: https://developers.google.com/workspace/guides/create-project
 - set up the environment per: https://developers.google.com/drive/api/quickstart/go#set_up_your_environment
   - App registration step:
     - App name: Synchronizer
     - Support email: <your email>
     - Developer contact information: <your email>
     - leave all else as default
   - Scopes step: add `.../auth/docs` ("See, edit, create, and delete all of your Google Drive files")
 - download the `client_secret_*.json` file to an appropriately secured directory, e.g.

```shell
mkdir ~/.google_api
chmod 700 ~/.google_api
mv ~/Downloads/client_secret_*.json ~/.google_api/client_secret.json
chmod 600 ~/.google_api/client_secret.json
```

## Usage

To download all documents from a Google Drive folder in markdown format run the following command:
```bash
./scripts/export.sh
```

To upload renderings of GitHub pages to Google Drive run the following command:
```bash
./scripts/export.sh
```
