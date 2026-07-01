# Connect to OpenVPN Server

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-open-vpn?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-open-vpn/releases)

Establish a VPN connection with the specified OpenVPN server.

<details>
<summary>Description</summary>

The Step establishes a VPN connection with the specified OpenVPN server.

### Configuring the Step

Before you start:
1. Start an OpenVPN server.
1. Register the contents of CA certificate, client certificate, client private key, base64 encoded as Bitrise Secrets.
   You can easily retrieve the contents of Base64 using command: `$ base64 <certificate or private key file path>`

To configure the Step:

1. Add the OpenVPN server IP or hostname to the **Host** input.
1. Add the OpenVPN server port number to the **Port** input. The port number is `1194` by default.
1. Specify the OpenVPN server protocol in the **Protocol** input.
1. Add the CA certificate, client certificate, and client private key Secrets to their respective inputs.

### Useful links

* [Using the Connect to OpenVPN Server Step](https://docs.bitrise.io/en/bitrise-platform/integrations/connecting-to-a-vpn-during-a-build#using-the-connect-to-openvpn-server-step)
* [Configuring network access for Bitrise build machines](https://docs.bitrise.io/en/infrastructure/build-machines/configuring-your-network-to-access-our-build-machines.html)

### Related Steps

* [Cisco VPN connect](https://www.bitrise.io/integrations/steps/vpnc-connect)

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `host` | Open VPN Server IP or Hostname. ex1. 111.111.111.111 ex2. hoge.com  | required |  |
| `port` | Open VPN Server Port number | required | `1194` |
| `proto` | Open VPN Server Protocol | required | `udp` |
| `ca_crt` | Base64 encoded CA Certificate | required, sensitive | `$VPN_CA_CRT_BASE64` |
| `client_crt` | Base64 encoded Client Certificate | required, sensitive | `$VPN_CLIENT_CRT_BASE64` |
| `client_key` | Base64 encoded Client Private Key | required, sensitive | `$VPN_CLIENT_KEY_BASE64` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `OPENVPN_LOG_PATH` | Path to the file containing the output (stdout and stderr) of the OpenVPN process. Useful for debugging connection issues.  |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-open-vpn/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-open-vpn/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
