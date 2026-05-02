$params = @{chat_id = "oc_1a243c33320b6e230f90c1a637e0e0a2"} | ConvertTo-Json
lark-cli.cmd im chat.members get --params $params --as bot
