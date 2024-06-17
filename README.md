## kubevirt qemu arg appender sidecar

Simple sidecar based on [kubevirt hook example.](kubevirt.io/kubevirt/cmd/example-hook-sidecar)

The sidecar appends qemu args to the end of the qemu commanline in libvirt xml.

That is it.

Ensure kubevirt feature gate `Sidecar` is enabled prior to use.

e.g. something like so if not:

```sh
[ ! kubectl get kubevirt -n harvester-system -o json | jq -r '.items[].spec.configuration.developerConfiguration.featureGates[]' | grep Sidecar ] && kubectl patch kubevirt -n harvester-system --type "json" -p '[{"op":"add","path":"/spec/configuration/developerConfiguration/FeatureGates/-","value":"Sidecar"}]'
```

Annotations needed to make this work are:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
...
spec:
  template:
    metadata:
      annotations:
        harvesterhci.io/sshNames: '[]'
        # Request the hook sidecar
        hooks.kubevirt.io/hookSidecars: '[{"image": "ghcr.io/mitchty/kubevirt-sidecar:main"}]'
        # Annotation with space delimited string of args to be added
        qemuargs.vm.kubevirt.io/args: -fw_cfg name=opt/ovmf/X-PciMmio64Mb,string=65536
```
