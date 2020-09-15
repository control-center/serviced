#!/bin/bash

pip install --upgrade pip
pip install --no-cache-dir -r $HOME/requirments.txt
pyinstaller -F elastic.py
