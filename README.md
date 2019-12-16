# duct

`duct` is derived/forked from [patchbay.pub](https://github.com/patchbay-pub/patchbay-simple-server). The name `duct` was chosen because this utility and server is meant to serve as a duct (or pipe) between utilities, and also as a sort of duct tape.

# What does it do?

`duct` enables IFTTT-type applications. The `duct` server provides infinite HTTP endpoints that provide a multi-process, multi-consumer (MPMC) queue which can be used to implement powerful serverless applications - including [desktop notificiations](#desktop-notifications), [sms notifications](#sms-notifications), [job queues](#job-queues), [web hosting](#web-hosting), and [file sharing](#file-sharing). These applications are basically just a few lines of bash that wrap a `curl` command. However, `duct` also contains a simple IMAP/SMTP for email reading and writing that facilitates a free and easy way to do SMS notifications and webhooks.

# Basics

The philosophy behind this is that the main logic happens on the local machine with small scripts. There is a server with an infinite number of virtual channels that will relay messages between the publisher and the subscriber. To subscribe to a channel you can simply make a `GET` request with a specified channel:

    curl https://duct.schollz.com/a61b1f42
    
The above will block until something is published to the channel `a61b1f42`. You can easily publish to a channel using a `POST` request:

    curl https://duct.schollz.com/a61b1f42 -d "hello, world"

The subscriber will immediately receive that data. If you reverse the order, then the post will block until it is received.

## Pubsub mode

The default mode for publishing and subscribing is a MPMC queue, where the first to connect are able to publish/subscribe. But you can also specify publish-subscribe (pubsub) mode. In pubsub mode, the publisher will become non-blocking and their data will be transmitted to each connected subscriber. This is done using the parameter `pubsub=true`:

    curl https://duct.schollz.com/a61b1f42?pubsub=true -d "hello, world"

## Publish with `GET`

You can also publish with a `GET` request by using the parameter `body=X`, so that the publisher statement can be also written as:

    curl https://duct.schollz.com/a61b1f42?pubsub=true&body=hello,%20world

This makes it easier to write href links that can trigger hooks.

# Examples 

## File sharing

Change the command on the sending computer to

    curl -X POST --data-binary "@test.txt" https://duct.schollz.com/test.txt
    
The command on the other computer remains the same as what you had

    wget https://duct.schollz.com/test.txt
    
### File sharing with end-to-end encryption + compresion

Sending:

    cat file | gzip | openssl bf -pbkdf2 | curl -X POST --data-binary @- https://duct.schollz.com/00bf8a8f-ded4-48ea-9409-43796e5b9587
  
To receive:

    curl --silent https://duct.schollz.com/00bf8a8f-ded4-48ea-9409-43796e5b9587 | openssl bf -pbkdf2 -d | gzip -d > file


## Web hosting 

```
while true; do curl -X POST https://duct.schollz.com/apps/simple_chat/chat.js --data-binary "@./apps/simple_chat/chat.js"; done
```

## Job Queues

```bash
#!/bin/bash

# IFS determines what to split on. By default it will split on spaces. Change
# it to newlines
# See https://www.cyberciti.biz/tips/handling-filenames-with-spaces-in-bash.html
ifsbak=$IFS
IFS=$(echo -en "\n\b")

for filename in *.mp3
do
        curl https://duct.schollz.com/6903f6bb-af5d-4ed9-98e7-3cdf1b5fa386 -d $filename
done

# Need to restore IFS to its previous value
IFS=$ifsbak
```

And on four other computers;

```bash
#!/bin/bash

while true
do
        filename=$(curl -s https://duct.schollz.com/6903f6bb-af5d-4ed9-98e7-3cdf1b5fa386)
        if [ "$filename" != "Too Many Requests" ]
        then
                echo $filename
                ffmpeg -i "$filename" "$filename.ogg"
        else
                sleep 1
        fi
done
```

## SMS notifications

First setup the email component of `duct`.

Then you can do a bash script like

```bash
#!/bin/bash

MAGIC="magic"
URL="https://duct.schollz.com/5de4210f-f835-4db2-bf42-5fe00bc0ae07"
CELLPHONE="XX@vtext.com"
while [ 1 ]
do
X="$(curl $URL)"
if [[ $X =~ ^$MAGIC ]]; then
        Y="$(echo "$X" | sed "s/$MAGIC*//")"
        duct email --to "$CELLPHONE" --subject "notify" --body "$Y"
else
        sleep 10
fi
done
```    


## Desktop notifications

### Windows

Run this PS script in the background. Follow [these instructions](https://stackoverflow.com/questions/20575257/how-do-i-run-a-powershell-script-when-the-computer-starts) to get this to work when you startup the computer.

```powershell
$magicprefix="magic"
$url="https://duct.schollz.com/03789265-d0c0-426b-b74d-d4d5f7f63a62"
while($true)
{
    $result = Invoke-WebRequest $url
    If ($result.StatusDescription -eq "OK") {
        if ($result.Content.StartsWith("$magicprefix") -eq $true) {
            Add-Type -AssemblyName System.Windows.Forms 
            $global:balloon = New-Object System.Windows.Forms.NotifyIcon
            $path = (Get-Process -id $pid).Path
            $balloon.Icon = [System.Drawing.Icon]::ExtractAssociatedIcon($path) 
            $balloon.BalloonTipIcon = [System.Windows.Forms.ToolTipIcon]::Warning 
            $content = $result.Content.TrimStart("$magicprefix")
            $balloon.BalloonTipText = "$content"
            $balloon.BalloonTipTitle = "Attention $Env:USERNAME" 
            $balloon.Visible = $true 
            $balloon.ShowBalloonTip(5000)        
        } else {
             Start-Sleep -s 1
        }
    } else {
        Start-Sleep -s 3
    }
}
```

### Linux

```bash
#!/bin/bash

MAGIC="magic"
URL="https://duct.schollz.com/405e3cda-7b36-4da4-9cd7-b46cab72c84b"

while [ 1 ]
do
X="$(curl $URL)"
if [[ $X =~ ^$MAGIC ]]; then
        Y="$(echo "$X" | sed "s/$MAGIC*//")"
        notify-send "$Y"
else
        sleep 10
fi
done
```    


### Email webhook

```
> duct email --check
{"to":"myemail@something.com","from":"someone@something.com","date":"2019-11-27T11:44:29-08:00","subject":"my subject","body":"my body"}
```

You can easily use `jq` or similar to process and add it to some sort of hook.

# License

MIT (with Copyright (c) 2019 patchbay)