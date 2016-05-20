#!/bin/bash
#
# Make sure there are no English translations that
# don't exist in Spanish, or removed keys that are
# still in the spanish file.
#

en_keys=`cat static/i18n/en_US.json | cut -d\" -f2 | sort`
es_keys=`cat static/i18n/es_US.json | cut -d\" -f2 | sort`

new_keys=`comm -23 <(echo "$en_keys" | sort) <(echo "$es_keys" | sort)`
removed_keys=`comm -13 <(echo "$en_keys" | sort) <(echo "$es_keys" | sort)`

if [[ ! -z $new_keys || ! -z $removed_keys ]]; then
  if [[ ! -z $new_keys ]]; then
    new_keys=`echo "$new_keys" | sed "s/.*/ &/" | paste -sd "," -`
    echo "These new keys must be added to the Spanish translation: ${new_keys}"
  fi
  if [[ ! -z $removed_keys ]]; then
    removed_keys=`echo "$removed_keys" | sed "s/.*/ &/" | paste -sd "," -`
    echo "These keys are in the Spanish translation but no longer in English: ${removed_keys}"
  fi
  exit 1
fi

