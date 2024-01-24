from invoke import task


@task
def build(c):
    tag = c.run("git describe --exact-match --tags", hide=True).stdout.strip()
    print("Tag was", tag)

    # Build universal mac executable.
    c.run("""GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o codecomet-amd64""")
    c.run("""GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o codecomet-arm64""")
    c.run("lipo -create -output codecomet codecomet-amd64 codecomet-arm64")
    c.run(f"zip -r codecomet-{tag}-osx-universal.zip ./codecomet")

    for os, nickname, arch, executable in [
        ("linux", "linux-x86_64", "amd64", "codecomet"),
        ("windows", "win64", "amd64", "codecomet.exe"),
    ]:
        c.run(
            f"""GOOS={os} GOARCH={arch} CGO_ENABLED=0 go build -ldflags="-w -s" -o {executable}"""
        )
        c.run(f"zip -r codecomet-{tag}-{nickname}.zip ./{executable}")
