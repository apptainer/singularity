# Singularity SCI-F Apps

Build your image

```
sudo singularity build cowsay.img Singularity.cowsay 
```

What apps are installed?

```
singularity apps cowsay.img
cowsay
fortune
lolcat
```

Ask for help for a specific app!

```
singularity help --app fortune cowsay.img
fortune is the best app
```

Run a particular app

```
singularity run --app fortune cowsay.img
```

Inspect an app

```
 singularity inspect --app fortune cowsay.img 
{
    "SINGULARITY_APP_NAME": "fortune",
    "SINGULARITY_APP_SIZE": "1MB"
}
```

