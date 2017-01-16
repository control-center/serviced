#! /bin/bash

# output, template, html, css, js
#   output: output file after all the things are done
#   template: template html that will be interpolated
#   html: html file to be injected into template
#   css: css file to be injected into template
#   js: js file to be injected into template

OUT=$1
IN=$2
export HTML=$(cat $3)
export CSS=$(cat $4)
export JS=$(cat $5)
echo "$(envsubst < $IN)" > $1
