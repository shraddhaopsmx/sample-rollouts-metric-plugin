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

You can also do this with many other debugers as well. Including cli debuggers like delve.

### Running the plugin
To run the plugin, you can run the following command you can start the argo rollouts controller with the plugin enabled
by setting the `--metric-plugin-location` flag on the rollouts controller to the path of the plugin binary. The flag
supports two schema's `file://` and `http(s)://`.

example:
```bash
./rollouts-controller --metric-plugin-location file://./metric-plugin
```