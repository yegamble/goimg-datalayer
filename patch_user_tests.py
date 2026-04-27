import re

content = open("tests/integration/user_repository_test.go").read()

# Pattern to find ReconstructUser and its arguments
pattern = r'(ReconstructUser\((?:[^)(]+|\([^)(]*\))*\))'

def repl(match):
    # This is a bit naive but should work if the call ends with &expiredTime or similar
    call = match.group(1)
    if "emailVerified" not in call and "false, nil" not in call:
        # replace the last `)` with `, false, nil)`
        return call[:-1] + ",\n\t\tfalse,\n\t\tnil,\n\t)"
    return call

# Specifically targeting expiredGuest
content = content.replace('&expiredTime,\n\t)', '&expiredTime,\n\t\tfalse,\n\t\tnil,\n\t)')
content = content.replace('&futureTime,\n\t)', '&futureTime,\n\t\tfalse,\n\t\tnil,\n\t)')


with open("tests/integration/user_repository_test.go", "w") as f:
    f.write(content)
