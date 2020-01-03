from os.path import exists


def main():
    print("Action Generator")

    name = input("Name: ")
    re = input("Regexp: ")

    actionName = name + "Action"
    newActionFunction = "new" + actionName[0].upper() + actionName[1:]

    actionsGo = "cmd/actions/actions.go"
    actionFile = "cmd/actions/" + name + ".go"

    addActionEntry = True

    if exists(actionFile):
        p = input("Action already exists do you want to overwrite it? [y/N] ").lower()
        if p not in ["y", "yes"]:
            return
        else:
            addActionEntry = False

    if addActionEntry:
        print("Adding Action to " + actionsGo)

        with open(actionsGo, "r") as f:
            lines = f.readlines()

        i = lines.index("var actions = Actions{\n")
        i = lines.index("}\n", i + 1)
        lines.insert(i - 1, "\t" + newActionFunction + "(),\n")

        with open(actionsGo, "w") as f:
            f.writelines(lines)

    print("Writing Action File: " + actionFile)
    with open(actionFile, "w") as f:
        f.write(
            f"""\
package actions

import (
    "regexp"
)

type {actionName} struct {{
    options *Options
}}

func {newActionFunction}() *{actionName} {{
    return &{actionName}{{
        options: &Options{{
            "Name": "{name}",
            "Re":   regexp.MustCompile(`{re}`),
        }},
    }}
}}

func (a {actionName}) GetOptions() *Options {{
    return a.options
}}

func (a {actionName}) Run(e *Event) error {{
    e.Say("{name}")

    return nil
}}
"""
        )


if __name__ == "__main__":
    main()
