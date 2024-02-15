from sys import platform
from invoke import task


@task
def build(c):
    tag = c.run("git describe --exact-match --tags", hide=True).stdout.strip()
    print("Tag was", tag)

    ldflags = f"-w -s -X 'github.com/codecomet-io/cli/cmd.CurrentVersion={tag}'"

    for os, nickname, arch, executable in [
        ("linux", "linux-amd64", "amd64", "codecomet"),
        ("linux", "linux-arm64", "arm64", "codecomet"),
        ("windows", "win64", "amd64", "codecomet.exe"),
        ("darwin", "macOS-amd64", "amd64", "codecomet"),
        ("darwin", "macOS-arm64", "arm64", "codecomet"),
    ]:
        c.run(
            f"""GOOS={os} GOARCH={arch} CGO_ENABLED=0 go build -ldflags="{ldflags}" -o {executable}"""
        )
        if os == "linux":
            c.run(f"tar -czvf codecomet-{tag}-{nickname}.tar.gz ./{executable}")
        else:
            c.run(f"zip -r codecomet-{tag}-{nickname}.zip ./{executable}")
