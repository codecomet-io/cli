from invoke import task


@task
def build(c):
    tag = c.run("git describe --exact-match --tags", hide=True).stdout.strip()
    print("Tag was", tag)

    for os, nickname, arch, executable in [
        ("linux", "linux-x86_64", "amd64", "codecomet"),
    ]:
        c.run(f"""GOOS={os} GOARCH={arch} go build -ldflags="-w -s" -o {executable}""")
        c.run(f"zip -r codecomet-{tag}-{nickname}.zip ./{executable}")
