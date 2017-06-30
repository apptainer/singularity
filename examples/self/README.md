# Bootstrap Self

A self bootstrap means packaging the current operating system that you live on into an image. Since we assume the root, you don't need to define a `From`. It looks like this:


## Options
```
Bootstrap: self
```

If you really wanted to specify some root, you could do this:

```
Bootstrap: self
From: /
```

And we highly recommend that you exclude paths that you don't want added to the tar. For example, Docker stores a lot of data in `/var`, so I chose to exclude that, along with some of the applications in `/opt`:

```
Bootstrap: self
Exclude: /var/lib/docker /home/vanessa /opt/*
```

## Build Example
so we could do the following with the specification build file in this folder:

```
singularity create --size 8000 container.img
sudo singularity bootstrap container.img Singularity
```


