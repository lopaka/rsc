Data Extraction Help and Cookbook
=================================

`rsc` supports the selection and extraction of data from an API response using a
[JSON:select](http://jsonselect.org) expression. [JSON:select](http://jsonselect.org) is very powerful but
it is also not the most intuitive language (unless you are intimately familiar with
CSS selectors) and the current library used in `rsc` leaves room for improvement...
Since API responses are all very similar to one another it is often easiest to create
a new query by looking up other similar queries. This is where this help page comes in.

Below are three sections:
 - A summary of [JSON:select](http://jsonselect.org) selectors
 - A definition for the `rsc` output formats
 - A cookbook with API 1.5 examples

JSON:Select selector summary
----------------------------

| Pattern | Meaning
| ------- | -------
| T | A node of type T, where T is one of: string, number, object, array, boolean, or null
| T.key | The value of a property called <key> that is the child of an object T
| T."complex key" | Same as previous, but with property name specified as a JSON string
| T:root | A node of type T which is the root of the JSON document
| T:nth-child(n) | A node of type T which is the nth child of an array parent (not supported)
| T:nth-last-child(n) | A node of type T which is the nth child of an array parent counting from the end (not supported)
| T:first-child | A node of type T which is the first child of an array parent (equivalent to T:nth-child(1))
| T:last-child | A node of type T which is the last child of an array parent (equivalent to T:nth-last-child(1))
| T:only-child | A node of type T which is the only child of an array parent
| T:empty | A node of type T which is an array or object with no child
| T U | A node of type U with an ancestor of type T
| T > U | A node of type U with a parent of type T
| T ~ U | A node of type U with a sibling of type T
| S1, S2 | Any node which matches either selector S1 or S2
| T:has(S) | A node of type T which has a child node satisfying the selector S
| T:expr(E) | A node of type T with a value that satisfies the expression E
| T:val(V) | A node of type T with a value that is equal to V
| T:contains(S) | A node of type T with a string value contains the substring S

`rsc` Output Formats
--------------------

### `--x1` -- single match

With `--x1` `rsc` expects to match a single element of the response JSON and outputs it
on stdout. In order to facilitate consumption by bash (or other unix shells) `rsc` outputs
the value as follows:
 - a null value results in no output
 - a numeric value is output as-is
 - a string value is output as-is
 - an array value is output as JSON on one line
 - a hash value is output as JSON on one line
In all cases no trailing newline is output.

Example:
```
$ rsc --host us-3.rightscale.com --key $RS_KEY --account 60073 --x1 ".name" cm15 show /api/clouds/6
EC2 us-west-2$
```
Note that the `$` prompt is on the same line as the response due to not having a newline.

A typical example might be:
```
$ name=`rsc --host us-3.rightscale.com --key $RS_KEY --x1 ".name" cm15 show /api/clouds/6`
```

### `--xm` -- multiple match

With `--xm` `rsc` outputs matches in JSON format one matched value per line. This makes it easy
to parse the result into an array in bash if you know the right trick.

Example (response truncated):
```
$ rsc --host us-3.rightscale.com --key $RS_KEY --xm ".name" cm15 index clouds
"EC2 us-east-1"
"EC2 us-west-1"
"AWS ap-southeast-1"
"AWS ap-northeast-1"
"EC2 us-west-2"
"EC2 sa-east-1"
"EC2 eu-west-1"
"EC2 ap-southeast-2"
[...]
$
```
Note that the quotes come from the JSON encoding of the string results.

This can be placed into a bash array as follows:
```
$ cloud_names=`rsc --host us-3.rightscale.com --key $RS_KEY --xm ".name" cm15 index clouds`
$ declare -a "cloud_names=($cloud_names)"
$ declare -p cloud_names
declare -a cloud_names='([0]="EC2 us-east-1" [1]="EC2 us-west-1" [2]="AWS ap-southeast-1" [3]="AWS ap-northeast-1" [...] )'
```
Note that it is possible to use `declare -a "cloud_names=(``rsc [...]``)"` as well.

The above array parsing works well for results that are strings or numbers. For results that
are more complex JSON values an alternative is to parse the results into an array line-by-line
as follows (beware, the quoting is essential):
```
$ ll="`rsc --host us-3.rightscale.com --key $RS_KEY --xm ".links" cm15 index clouds`"
$ echo $ll | wc -l
9
$ readarray -t links <<<"$ll"
$ declare -p links
declare -a ll='([0]="[{\"href\":\"/api/clouds/1\",\"rel\":\"self\"},...]"
[1]="[{\"href\":\"/api/clouds/3\",\"rel\":\"self\"},{\"href\":\"/api...]"
[2]="[{\"href\":\"/api/clouds/4\",\"rel\":\"self\"},{\"href\":\"/api...
...
```
Note: the output of declare is manually pretty-printed and abridged, the real output is not
broken up in lines, unfortunately.

### `--xj` -- multiple match JSON output

With `--xj` `rsc` outputs an array of JSON matches. This is only really useful when
the consumer of the output can parse JSON. Example:
```
$ rsc --host us-3.rightscale.com --key $RS_KEY --x1 ".name" cm15 show /api/clouds/6
["EC2 us-east-1","EC2 us-west-1","AWS ap-southeast-1",...]
```

`json` Subcommand
-----------------

In some cases it may be necessary to extract multiple values from a single response in multiple
passes. `rsc` exposes a `json` command for that purpose which reads the json from STDIN and outputs
the result of applying the jsonselect expression specified via the `--x1` or `--xm` flags to STDOUT:
```
$ echo '{"foo":"bar"}' | rsc --x1=.foo json
bar
$ rsc cm15 index clouds | rsc --xm=.name json
"EC2 us-east-1"
"Openstack Havana"
"OpenStack Grizzly"
"CS 4.2.1 - KVM"
"Rackspace Open Cloud - Chicago"
"Google"
"Azure West US"
"SoftLayer"
"VScale Engineering v5.5"
```

Cookbook
--------

- Find instance's public IP addresses
```
$ rsc --host us-3.rightscale.com --key 1234567890 \
           --x1 '.public_ip_addresses' cm15 show /api/clouds/1/instances/LAB4OFL7I82E
["54.147.25.88"]
```

- Find an instance's resource_uid:
```
rsc --host us-3.rightscale.com --key 1234567890 \
           --x1 '.resource_uid' cm15 show /api/clouds/1/instances/LAB4OFL7I82E
"i-4e9a80b5"
```

- Find an instance's server href:
```
$ rsc --host us-3.rightscale.com --key 1234567890 \
           --x1 'object:has(.rel:val("parent")).href' \
           cm15 show /api/clouds/1/instances/LAB4OFL7I82E
"/api/servers/994838003"
```

- Find an instance's cloud type:
```
cloud=$(rsc --host us-3.rightscale.com --key 1234567890 \
        --x1 'object:has(.rel:val("cloud")).href' cm15 show /api/clouds/1/instances/LAB4OFL7I82E)
rsc --host us-3.rightscale.com --key 1234567890 \
         --x1 '.cloud_type' cm15 show $cloud
```

- Find the hrefs of all clouds of type amazon:
```
$ rsc --host us-3.rightscale.com --key 1234567890 \
           --xm 'object:has(.rel:val("self")).href' cm15 index clouds 'filter[]=cloud_type==amazon'
"/api/clouds/1"
"/api/clouds/3"
"/api/clouds/4"
"/api/clouds/5"
"/api/clouds/6"
"/api/clouds/7"
"/api/clouds/2"
"/api/clouds/8"
"/api/clouds/9"
```
Note: the match `object:has(.rel:val("self")).href` serves to extract the hrefs from the _self_
links. The returned JSON for each cloud includes
`"links":[ {"href":"/api/clouds/7", "rel":"self"}, {"href":"/api/clouds/7/datacenters",
"rel":"datacenters"}, ... ]` and the [JSON:select](http://jsonselect.org) expression says:
find an _object_ (JSON hash) that has a _rel_ child/field whose value is _self_
and then extract the value of the _href_ child/field. The _object_ here matches the
`{"href":"/api/clouds/7","rel":"self"}` hash.

Illustrating the difference between `--x1`, `--xm`, and `--xj`:
- `--x1` produces: `rsc: error: Multiple values selected, result was:
  <<[{"cloud_type":"amazon","descr... >>` with a non-zero exit code (it prints the raw JSON
	for troubleshooting purposes).
- `--xm` produces: `"/api/clouds/1" "/api/clouds/3" "/api/clouds/4" "/api/clouds/5"
  "/api/clouds/6" "/api/clouds/7" "/api/clouds/2" "/api/clouds/8" "/api/clouds/9"` and can be used
	in bash as `cloud_hrefs=$(rsc ...)`
- `--xj` produces: `["/api/clouds/1", "/api/clouds/3", "/api/clouds/4", "/api/clouds/5",
   "/api/clouds/6", "/api/clouds/7", "/api/clouds/2", "/api/clouds/8", "/api/clouds/9"]`

Find a running or stopped instance by public IP address in AWS us-east (cloud #1):
```
$ rsc --host us-3.rightscale.com --key 1234567890 \
           --xm 'object:has(.rel:val("self")).href' cm15 index /api/clouds/1/instances \
           'filter[]=public_ip_address==54.147.25.88' 'filter[]=state<>terminated' \
           'filter[]=state<>decommissioning' 'filter[]=state<>terminating' \
           'filter[]=state<>stopping' 'filter[]=state<>provisioned' 'filter[]=state<>failed'
"/api/clouds/1/instances/LAB4OFL7I82E"
```

