import os

import matplotlib.pylab as plt
import seaborn as sns

from compatlib.logger import logger
from compatlib.traces import TraceSet


def main(args, parser, extra, subparser):
    """
    The "extra" here is the list of events

    compatlib analyze-recording $(find ../recording -name *.out)
    """
    # Extra events here should be one or more result event files to parse
    events = extra
    total_events = len(events)

    # An output directory is required
    if not args.outdir:
        logger.exit("Please specify an output directory with -d/--outdir")

    # A trace set is a collection of event files
    traceset = TraceSet(events)
    df = traceset.to_dataframe()

    # Define output files and paths
    image_outdir = os.path.join(args.outdir, "img")
    events_csv = os.path.join(args.outdir, "events-dataframe.csv")
    if not os.path.exists(image_outdir):
        os.makedirs(image_outdir)

    # Tell the user where the stuff is going!
    logger.info(f"Output will be saved to: {args.outdir}")
    logger.info(f"              🎨 Images: {image_outdir}")
    logger.info(f"              ⏲️ Events: {events_csv}")

    df.to_csv(args.event_csv)

    # This is a distance matrix
    sims = traceset.distance_matrix()
    print(sims)

    # Clean up release names
    sims.index = [x.replace(args.suffix, "") for x in sims.index]
    sims.columns = [x.replace(args.suffix, "") for x in sims.columns]
    if args.prefix is not None:
        sims.index = [x.replace(args.prefix, "") for x in sims.index]
        sims.columns = [x.replace(args.prefix, "") for x in sims.columns]

    plt.figure(figsize=(20, 20))
    sns.clustermap(sims.astype(float), mask=(sims == 0.0), cmap="tab20b")

    # Save all the things!
    plot_path = os.path.join(args.outdir, f"{args.name}-levenstein-distance-matrix.png")
    title = (
        f"Levenstein Distance of File Access for {total_events} Recorded {args.name} Release Runs"
    )
    plt.title(title.rjust(200), fontsize=10)
    plt.tight_layout()
    plt.xlabel("Release Tag")
    plt.ylabel("Release Tag")
    plt.savefig(plot_path)
    plt.close()
