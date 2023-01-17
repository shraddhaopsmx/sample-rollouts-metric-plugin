# sample-rollouts-metric-plugin
This contains an example plugin for use with Argo Rollouts plugin system

### Build

To build a release build run the command below:
```bash
make build-sample-plugin
```

To build a debug build run the command below:
```bash
make build-sample-plugin-debug
```

### Attaching a debugger to debug build
If using goland you can attach a debugger to the debug build by following the directions https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html

You can also do this with many other debuggers as well. Including cli debuggers like delve.

### Running the plugin
To run the plugin, you can run the following command you can start the argo rollouts controller with the plugin enabled
by setting the `--metric-plugin-location` flag on the rollouts controller to the path of the plugin binary. The flag
supports two schema's `file://` and `http(s)://`.

example:
```bash
./rollouts-controller --metric-plugin-location=file://./metric-plugin
```

or via http(s)

There are however downsides to using a url such as having to download the binary every time the controller starts up and/or restarts. If the service hosting the binary is down
the controller will also not be able to start which could induce downtime. To work around those we suggest hosting the plugin internally and mouting the file up via k8s
configuration.
```bash
./rollouts-controller --metric-plugin-location=https://github.com/argoproj-labs/sample-rollouts-metric-plugin/releases/download/v0.0.1/metric-plugin-linux-amd64
```

### Sample Analysis Template
When configuring a AnalysisTemplate `provider.plugin.config:` can be anyhing you need it to be and it will be passed into the the plugin via the Metric struct.

An example for this sample plugin below:
```
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
    - name: service-name
  metrics:
    - name: success-rate
      interval: 5s
      # NOTE: prometheus queries return results in the form of a vector.
      # So it is common to access the index 0 of the returned array to obtain the value
      successCondition: result[0] >= 8
      failureLimit: 2
      count: 3
      provider:
        plugin:
          config:
            address: http://prometheus.local
            step: 1m
            query: |
              machine_cpu_cores
```
