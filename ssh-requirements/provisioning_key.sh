#!/bin/bash

# CRED is the variable populated from build parameter
function get_credentials() 
{
   if [ -n "$CRED" ]
   then
     # for demonstration purposes (this is not 100% secure),
     # credentials are passed via build parameter as a plain text
    echo "$CRED"

    ## in a strategy based on secrets server, CRED is a temporary access token
    ## this expirable token is used to obtain actual credentials string via REST API
    # curl "https://my.secrets.server/bundler_credentials?token=$CRED"

    ## in a strategy based on FS squashing, CRED is a path to secrets file
    # cat “$CRED” && rm -f “$CRED”
  fi
}


CREDENTIALS_STRING="$( get_credentials )"

if [ -n "$CREDENTIALS_STRING" ] 
then
    # /dev/shm is a temporary, memory-mapped FS
    mkdir -p /tmp/
    mkdir -p /home/atlantis/.ssh
    TEMPFILE=/tmp/deployment.key
    echo "$CREDENTIALS_STRING" > $TEMPFILE
    cp /tmp/deployment.key /home/atlantis/.ssh/id_rsa
    chmod 0600 $TEMPFILE
    chmod 0600 /home/atlantis/.ssh/id_rsa
    chown atlantis.atlantis /home/atlantis/.ssh/id_rsa
    eval $(ssh-agent)
    ssh-add $TEMPFILE
    ssh-add /home/atlantis/.ssh/id_rsa
    ssh-keyscan -t rsa github.com >> /home/atlantis/.ssh/known_hosts
    chown atlantis.atlantis /home/atlantis/.ssh/known_hosts
    #rm $TEMPFILE
fi
