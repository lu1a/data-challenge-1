#!/bin/bash

REPO_NAME="data-challenge-1"

REPO_FOLDER="/root/${REPO_NAME}"
RELEASE_FOLDER="${REPO_FOLDER}"

NGINX_CONFIG_FILE="/etc/nginx/sites-available/data-challenge-1.lu1.in"
DESIRED_PORT=4000
SYSTEMD_CONFIG_FOLDER="/etc/systemd/system"

LAST_DEPLOYED_COMMIT=""
LATEST_COMMIT=""

get_last_deployed_commit() {
    file_path=$(find $RELEASE_FOLDER -type f -name "${REPO_NAME}-*" -print -quit)

    if [[ -n "$file_path" ]]; then
        filename=$(basename "$file_path")
        LAST_DEPLOYED_COMMIT="${filename##*-}"

    else
        echo "No previous deployment found."
    fi
}

get_latest_commit() {
    git -C $REPO_FOLDER pull
    LATEST_COMMIT=$(git -C $REPO_FOLDER rev-parse HEAD)
}

build_stage() {
    echo "Running build stage"

    /usr/local/go/bin/go build -C $REPO_FOLDER -o $REPO_NAME
    mv $RELEASE_FOLDER/$REPO_NAME $RELEASE_FOLDER/$REPO_NAME-$LATEST_COMMIT
    chmod 700 $RELEASE_FOLDER/$REPO_NAME-$LATEST_COMMIT
}

deploy_stage() {
    echo "Running deploy stage"

    # Use grep and awk to find the proxy_pass directive and extract the port number
    proxy_pass_port=$(grep -E '^\s*proxy_pass\s+http://127.0.0.1:([0-9]+);' "$NGINX_CONFIG_FILE" | awk -F':' '{print $NF}')
    proxy_pass_port="${proxy_pass_port%;}"

    # increment one port up for the new service, or wrap back to original desired port on the third increment
    new_port="$((((proxy_pass_port + 1) % 3) + DESIRED_PORT))"

    echo "Current port: $proxy_pass_port"
    echo "Toggled port: $new_port"

    # Create and run a new service for the newly built commit
    new_service_contents=$(cat <<EOF
[Unit]
Description=$REPO_NAME service
After=network.target
StartLimitIntervalSec=0
[Service]
Type=simple
Restart=always
RestartSec=1
User=root
ExecStart=$RELEASE_FOLDER/$REPO_NAME-$LATEST_COMMIT 127.0.0.1:$new_port $MY_REAL_TELEGRAM_USERNAME $REPO_FOLDER

[Install]
WantedBy=multi-user.target
EOF
    )
    echo "$new_service_contents" > $SYSTEMD_CONFIG_FOLDER/$REPO_NAME-$LATEST_COMMIT.service
    systemctl start $REPO_NAME-$LATEST_COMMIT
    systemctl enable $REPO_NAME-$LATEST_COMMIT

    # Wait 5 seconds for the new service to properly start up
    sleep 5

    sed -i "s|^\([[:space:]]*proxy_pass[[:space:]]\+http://127.0.0.1:\)[0-9]\+;|\1$new_port;|" "$NGINX_CONFIG_FILE"
    nginx -t && nginx -s reload

    previous_deployments=( "$SYSTEMD_CONFIG_FOLDER"/$REPO_NAME-*.service )

    # Loop over the files, excluding the one with the specific commit hash
    for file in "${previous_deployments[@]}"; do
        servicename=$(basename "$file")
        if [[ "$servicename" != "$REPO_NAME-$LATEST_COMMIT.service" ]]; then
            echo "Deleting previous deployment: $servicename"

            commit_hash="${servicename#$REPO_NAME-}"
            commit_hash="${commit_hash%.service}"
            systemctl disable $REPO_NAME-$commit_hash
            systemctl stop $REPO_NAME-$commit_hash
            rm $SYSTEMD_CONFIG_FOLDER/$REPO_NAME-$commit_hash.service

            rm $RELEASE_FOLDER/$REPO_NAME-$commit_hash
        fi
    done
    systemctl daemon-reload
}

get_last_deployed_commit
get_latest_commit

if [[ "$LATEST_COMMIT" != "$LAST_DEPLOYED_COMMIT" ]]; then
    echo "$(date --utc +%Y-%m-%dT%H:%M:%SZ) -- A new change has been pushed"

    build_stage
    deploy_stage
fi
