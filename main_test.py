import pytest

from main import replace


@pytest.mark.parametrize("codeowners, src, dest, expected", [
    pytest.param("* @a", "a", "b", "* @b", id="a to b"),
    pytest.param("* @a @b", "a", "b", "* @b", id="merged forward"),
    pytest.param("* @b @a", "a", "b", "* @b", id="merged backward"),
    pytest.param("* @a @c @b", "a", "b", "* @b @c", id="merged to source"),
    pytest.param("* @b @c @a", "a", "b", "* @b @c", id="merged to boss"),
    pytest.param("* @a/a @a @b" , "a/a", "b", "* @b @a", id="team a to b"),
    pytest.param("* @a @aa", "a", "b", "* @b @aa", id="keep order"),
    pytest.param("*    @a", "a", "b", "*    @b", id="keep whitespace"),
    pytest.param("*\t@a  @b\t\t@c", "a", "b", "*\t@b  @c", id="keep all kind whitespaces"),
    pytest.param("* @a ", "a", "b", "* @b", id="remove trailling whitespace"),
    pytest.param("* @a\na @a @b", "a", "b", "* @b\na @b", id="multiline"),
    pytest.param("# codeowners\n* @a\n\n", "a", "b", "# codeowners\n* @b\n\n", id="ignore non rule line"),
    pytest.param("* @a\n# .github @a", "a", "b", "* @b\n# .github @a", id="ignore commented line"),
    pytest.param("* @a\n/example\\ path/ @a", "a", "b", "* @b\n/example\\ path/ @b", id="keep whitespace path name"),
])
def test_replace(codeowners, src, dest, expected):
    got = replace(codeowners, src, dest)
    assert got == expected
