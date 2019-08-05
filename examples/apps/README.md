# Singularity SCI-F Apps

The Scientific Filesystem is well suited for Singularity containers to allow you
to build a container that has multiple entrypoints, along with modular environments,
libraries, and executables. Here we will review the basic building and using of a
Singularity container that implements SCIF. For more quick start tutorials, see
the [official documentation for SCIF](https://vsoch.github.io/scif/).

Build your image

```
sudo singularity build cowsay.sif Singularity.cowsay 
```

What apps are installed?

```
singularity apps cowsay.sif
cowsay
fortune
lolcat
```

Ask for help for a specific app!

```
singularity help --app fortune cowsay.sif
fortune is the best app
```

Run a particular app

```
singularity run --app fortune cowsay.sif
When I reflect upon the number of disagreeable people who I know who have gone
to a better world, I am moved to lead a different life.
		-- Mark Twain, "Pudd'nhead Wilson's Calendar"
```

Inspect an app

```
 singularity inspect --app fortune cowsay.sif 
{
    "SCIF_APPNAME": "fortune",
    "SCIF_APPSIZE": "1MB"
}
```
