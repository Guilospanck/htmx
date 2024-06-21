# htmx

[the docs](https://htmx.org/docs/#introduction)

- any element can be targetted to have its contents updated, not just the window
- any element can issue HTTP requests
- on the server side you respond with HTML

_"So, in summary, all you need to do to use CSS transitions for an element is keep its id stable across requests!"_ --From [docs](https://htmx.org/docs/#css_transitions)

From [docs]:

> _"To understand how CSS transitions actually work in htmx, you must understand the underlying swap & settle model that htmx uses.
> When new content is received from a server, before the content is swapped in, the existing content of the page is examined for elements that match by the id attribute. If a match is found for an element in the new content, the attributes of the old content are copied onto the new element before the swap occurs. The new content is then swapped in, but with the old attribute values. Finally, the new attribute values are swapped in, after a “settle” delay (20ms by default). A little crazy, but this is what allows CSS transitions to work without any javascript by the developer."_

- [Out of band swap](https://htmx.org/attributes/hx-swap-oob/) can be used to inform a part of the code that something happened (think like a toast).

If there is content that you wish to be preserved across swaps (e.g. a video player that you wish to remain playing even if a swap occurs) you can use the [hx-preserve](https://htmx.org/attributes/hx-preserve/) attribute on the elements you wish to be preserved.
