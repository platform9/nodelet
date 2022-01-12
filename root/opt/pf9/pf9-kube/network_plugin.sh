if [[ -z "$PF9_NETWORK_PLUGIN" ]]; then
    echo "Network plugin not set"
    exit 1
fi

source network_plugins/"$PF9_NETWORK_PLUGIN"/"$PF9_NETWORK_PLUGIN".sh

## Required methods defined by each network plugin
# network_running - Function that health checks the app backing the selected network plugin
# ensure_network_running - Start/Restart the app backing the selected network plugin
# ensure_network_config_up_to_date - Any master only configuration that the plugin will need to write/use
# write_cni_config_file - Write CNI config file to $CNI_CONFIG_DIR that K8S will use. CNI type will be specific to the plugin
# ensure_network_controller_destroyed - Stop/Kill the app backing the selected network plugin and clean up state.
