# Using tedge-container-plugin with podman rootless

The following page includes some hints on how to setup podman in rootless mode so that the tedge user can run commands.

If you run into any errors please consult the official container engine's documentation.

## Alpine Linux (with OpenRC)

1. Create a home folder for the tedge user (required by podman)

    ```sh
    sudo apk --no-cache add shadow
    sudo mkdir -p /home/tedge/.config/containers/
    sudo chown -R tedge:tedge /home/tedge
    sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 tedge
    ```

    If you need/want to avoid installing the `shadow` package (which provides the `usermod` command) then you will have to manually modify the `/etc/subuid` and `/etc/subgid` files. below shows an example of how to do this.

    ```sh
    echo tedge:100000:165535 | sudo tee -a /etc/subuid
    echo tedge:100000:165535 | sudo tee -a /etc/subgid
    ```

    **Note:** Technically it would be feasible to edit the default storage path in the container.conf file, however that change might be more invasive for other users who wish to run their own podman rootless containers.

2. Edit the podman OpenRC service to run the api endpoint (socket) under a non-root user

    ```sh
    sudo sed -i 's/.*podman_user=.*/podman_user="tedge"/g' /etc/conf.d/podman
    ```

3. Restart the podman api service

    ```sh
    sudo tedgectl restart podman
    ```

    **Note:** You can also run the openrc specific commands, `tedgectl` just makes it easier and is service manager agnostic.

4. Edit the tedge-container service to run using the tedge user (instead of root)

    **OpenRC**

    Change the `command_user` to use `tedge`

    ```sh
    sudo sed -i 's/.*command_user=.*/command_user="tedge"/g' /etc/conf.d/tedge-container-plugin
    ```

5. Restart the tedge-container-plugin service

    ```sh
    sudo tedgectl restart tedge-container-plugin
    ```

    **Note:** You can also run the openrc specific commands, `tedgectl` just makes it easier and is service manager agnostic.
