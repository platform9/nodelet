#@IgnoreInspection BashAddShebang

if [ -f /etc/pf9/kube_override.env ]; then
    source /etc/pf9/kube_override.env
fi
if [ "$RUNTIME" == "containerd" ]; then
    source containerd_runtime.sh
else
    source docker_runtime.sh
fi