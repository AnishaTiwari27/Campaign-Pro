# Decisions worth knowing

Choices that look arbitrary from the outside and are not. Each of these
has bitten, or would have.

### Photographs of people

`creators.image_url` holds a person's own photograph, and `SubjectImage`
prefers it over the generated avatar wherever that person appears — the
roster, their detail page, and every campaign tile they front. A failed
load falls back to the avatar, so a dead URL degrades rather than breaks.

```sql
UPDATE creators SET image_url = 'https://…/aanya.jpg' WHERE id = 'aanya-verma';
```

An empty column is the normal state, not a gap to be filled. The generated
avatar is the correct likeness for someone whose photo the agency has no
right to publish, which is most people most of the time.

Two things not to put in that column. A real celebrity's photograph beside
these names attaches invented spend, reach and sign-off to someone who
never ran the campaign — the creators in the demo data are fictional, and
the pairing would be a fabricated endorsement. Stock portraits carry the
same problem in licence form: most stock licences forbid using a model's
likeness to imply they endorse a product. Synthetic portraits of people
who do not exist, or a real creator's own photo used with their
permission, are the two that work.

### A word on "creators" vs "Influencers"

Half the campaigns here are fronted by a person rather than a brand — in the
demo data that is cricketers, singers, actors and creators, 18 of 35. The UI
calls that section **Influencers**, because that is the market it belongs to
and because an abstract label made people guess.

The route (`/creators`), the API path, the Go package and the `creators`
table all keep the original name. Renaming them would break every link
already shared and change nothing a reader can see. The divergence is
deliberate and noted at the nav item itself.

### Self-registration

`ALLOW_SIGNUP` (on by default) puts a **Create an account** link on the sign-in
screen, and serves `POST /api/auth/signup`. `/#signup` opens the form directly,
so an invite mail can link straight to it.

A self-registered account is always an **analyst**: agency-side, reads
campaigns, analytics and reports, and decides nothing. No approvals, no user
management, no settings. The role is a constant in
[`signup.go`](backend/services/identity/internal/service/signup.go) rather than
anything the request can influence, so the page cannot be used to mint an
admin. Promotion is a deliberate act, and today that means `createuser`.

Signup deliberately does **not** issue a session. The new account is sent to
the sign-in form to use the password it just chose, which proves the password
works before anything depends on it, and keeps the endpoint from being a way to
mint sessions.

Campaigns owns the `users` table, so identity does not write it: signup asks
for the capability through a `Directory` interface that campaigns supplies.
Neither service imports the other — the same shape as the existing `Auditor`.

Note what the duplicate-email case gives away. A signup form that rejects a
taken address tells the caller that address is registered; the alternative is
accepting a duplicate silently and leaving someone with no account and no
explanation. Sign-in, which has no such excuse, stays deliberately vague about
whether an email exists.

## Ad films are embedded from the advertiser's own YouTube channel

The spotlight plays the real ad film a campaign is running, where one
can be verified. Every entry in `frontend/src/data/adFilms.ts` is a
video the brand published on its own channel and left embeddable.

That is the whole licensing argument: the player is YouTube's, the
upload is the advertiser's, attribution travels with it, and nothing is
copied or re-hosted. It is what any publisher does when it embeds a
brand's ad.

Three things are refused, and the refusals are the point:

- creative pulled out of an ad server, which is not ours to serve;
- a "banner" composited from a celebrity photograph and a brand logo,
  which fabricates an advertisement that does not exist and attaches it
  to a real person;
- a film from a fan channel that re-uploaded it, which looks fine while
  attributing a stranger's upload to the advertiser.

So `channel` is stored beside every video, and
`scripts/verify-ad-films.mjs` re-checks it against YouTube's oEmbed
endpoint rather than trusting the file. oEmbed answers all three failure
modes at once: 404 for a deleted video, 401/403 for one that may not be
embedded, and the publishing channel's name. It caught a wrong title on
its first run.

A campaign with no verified film shows what it showed before. A gap is
not a thing to fill with something approximate.

The player is a facade — a still and a play button until clicked, then
`youtube-nocookie.com`. On a dashboard most visitors never press play,
and an embed that mounts on load would phone home and set cookies for
all of them.
