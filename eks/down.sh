#!/bin/bash

aws fsx describe-file-systems --query "FileSystems[?contains(Name, 'alphafold')].FileSystemId" --output text | xargs -I {} aws fsx delete-file-system --file-system-id {} --force