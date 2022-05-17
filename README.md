# AWS IoT secure tunneling localproxy implementation
- This is a repository created for studying for rehabilitation.
- I do not speak English, so corrections in English are welcome.

## Reference
- [AWS documents](https://docs.aws.amazon.com/iot/latest/developerguide/secure-tunneling.html)

- [V2WebSocketProtocolGuide.md](https://github.com/aws-samples/aws-iot-securetunneling-localproxy/blob/v2.1.0/V2WebSocketProtocolGuide.md)


## localproxy golang implementation
### Build

- You have docker environment.

    ``` sh
    $ git clone https://github.com/mizosukedev/securetunnel
    $ cd securetunnel

    $ ./misc/docker/run_on_docker.sh make build
    # or
    $ ./misc/docker/run_on_docker.sh make
    ```

- You have golang environment.

    ``` sh
    $ cd "${GOPATH}/src"
    $ git clone https://github.com/mizosukedev/securetunnel
    $ cd securetunnel
    
    $ make build
    # or
    $ make
    ```

### Usage

#### Forward proxy
- Setup forward proxy, if you need. The forward proxy can be specified with the `http_proxy` and `https_proxy` environment variables.

- Exxample

    ``` sh
    $ proxy=http://[proxy host address]:[proxy port]
    $ export https_proxy=${proxy}
    $ export http_proxy=${proxy}
    ```

#### Source mode
- Exmample:

    ``` sh
    $ export MITRA_LOCALPROXY_TOKEN=<set source token>

    # If destinationConfig.services are not specified when executing OpenTunnel
    $ mitra_localproxy -e "wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel" -s 10022

    # If services are specified, specify the service "service name=[address:]port", separated by commas.
    $ mitra_localproxy -e "wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel" -s "SSH=10022, RDP=13389"
    ```

#### Destination mode
- Example:

    ``` sh
    $ export MITRA_LOCALPROXY_TOKEN=<set destination token>

    # If services are not specified when executing OpenTunnel
    $ mitra_localproxy -e wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel -d 22

    # If services are specified, specify the service "service name=[address:]port", separated by commas.
    $ mitra_localproxy -e wss://data.tunneling.iot.us-east-1.amazonaws.com:443/tunnel -d "SSH=22, RDP=3389"
    ```

## Server (TBD)
