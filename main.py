import time
from collections import Counter
from os import listdir
from os.path import isfile, join

BEFORE_PATH = "input/org"
AFTER_PATH = "output/org"
DIFF_PATH = "diff/org"

PATH = "output/org"


def replace(content: str, src: str, dest: str) -> str:
    c = "\n".join([replace_line(l, src, dest) for l in content.splitlines()])
    if content.endswith("\n"):
        c += "\n"
    return c


def replace_line(line: str, src: str, dest: str) -> str:
    if "@" not in line or line.startswith("#") or src not in line:
        return line

    already = set()
    target, *owners = line.split("@")
    tt = [target]
    for name in owners:
        n = name.strip()

        if n != src and n not in already:
            already.add(n)
            tt.append(name)
            continue

        name = name.replace(src, dest)
        n = name.strip()
        if n in already:
            continue

        if n not in already:
            already.add(n)

        tt.append(name)

    tt[-1] = tt[-1].strip()
    return "@".join(tt)


def analyze():
    files = [f for f in listdir(PATH) if isfile(join(PATH, f))]

    c = Counter()

    for f in files:
        with open(join(PATH, f)) as fp:
            for l in fp:
                if l.strip() == "":
                    continue
                if "@" not in l:
                    continue
                if "#" in l:
                    continue
                owners = l.split()[1:]
                c.update(owners)
    print(list(c))


def main():
    log_fp = open(f"operation-{time.time()}.log", "w")
    logs = []
    files = [f for f in listdir(PATH) if isfile(join(PATH, f))]
    for f in files:
        with open(join(PATH, f), "r+") as fp:
            old = fp.read()
            fp.seek(0)
            new = replace(old, "a", "b")
            if old != new:
                log = f"=====\n{join(PATH, f)}\n{old}\n-----\n{new}\n"
                log_fp.write(log)
                logs.append(log)
            fp.write(new)
            fp.truncate()
    log_fp.close()
    print("\n".join(logs))


def diff():
    files = [f for f in listdir(BEFORE_PATH) if isfile(join(BEFORE_PATH, f))]
    for f in files:
        with open(join(BEFORE_PATH, f)) as bf, open(join(AFTER_PATH, f)) as af:
            old, new = bf.read(), af.read()
            if old != new:
                with open(join(DIFF_PATH, f), "w") as df:
                    log = f"=====\n{join(PATH, f)}\n{old}\n-----\n{new}\n"
                    print(log)
                    df.write(new)


if __name__ == "__main__":
    diff()
