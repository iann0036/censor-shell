# censor-shell

![](https://raw.githubusercontent.com/iann0036/iann0036/censor-shell.gif)

## Installation

```
go install
```

## Usage

Make the file `~/.censor-shell` as an INI file with the following content:

```
[nameofmyreplacement]
pattern = badword
replacement = goodword

[anotherpattern]
pattern = abc([a-z]+)ghi
replacement = zyx${1}tsr
```

You can specify any amount of replacement rules as you like. Patterns and replacement follow standard Go [regexp](https://golang.org/pkg/regexp/) formats.

Now open a new shell and execute the `censor-shell` command. You'll be able to see that all outputs are replaced dynamically:

```
> echo badword
goodword
> echo abcdefghi
zyxdeftsr
```
