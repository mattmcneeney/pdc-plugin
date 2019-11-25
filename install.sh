#!/bin/bash -x

go build && cf uninstall-plugin pdc-plugin
cf install-plugin pdc-plugin

