#!/bin/bash

function has_apt_get {
  [[ -n "$(command -v apt-get)" ]]
}

if $(has_apt_get); then
  # Ubuntu does not have pip pre-installed, Amazon Linux does
  sudo apt-get install -y python-pip
  export LC_ALL="C" # Necessary for running pip install
fi

sudo pip install boto3
