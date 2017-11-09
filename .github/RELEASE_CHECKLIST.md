# Release Checklist


## Code
- [] Ensure docs exist at singularityware.github.io for new/updated features
- [] Update version number in `configure.ac`
- [] Update changelog and version in `debian/changelog`
- [] Confirm tests exist for new features, and tests pass
- [] Commit the changes: 

```
git add .
git commit -m "Release 2.3.3"
```

## Github

- [] Tag the last git commit with the version number:

```
git tag -a 2.3.3
```

## Announcement
- [] Initial announcement to list (@gmk)
- [] Announcement for singularityware.github.io (ensure links work) (@vsoch)
- [] If release, @SingularityWare on twitter
