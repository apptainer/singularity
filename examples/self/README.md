# Bootstrap Self

A self bootstrap means packaging the current operating system that you live on into an image. Since we assume the root, you don't need to define a `From`. It looks like this:

```
Bootstrap: self
```

so we could do the following with the specification build file in this folder:

```
singularity create --size 8000 container.img
sudo singularity bootstrap container.img Singularity
```


