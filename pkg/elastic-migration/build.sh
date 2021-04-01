#!/bin/bash

pip install --upgrade pip==20.3.4
pip install --no-cache-dir -r $HOME/requirments.txt
pyinstaller -F elastic.py
