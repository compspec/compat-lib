import os

from compatlib.logger import logger
from compatlib.traces import TraceSet


def main(args, parser, extra, subparser):
    """
    The "extra" here is the list of events

    compatlib to-perfetto $(find ../recording -name *.out)
    """
    # Extra events here should be one or more result event files to parse
    events = extra

    # An output directory is required
    if not args.outdir:
        logger.exit("Please specify an output directory with -d/--outdir")

    # Define output files and paths
    outfile = os.path.join(args.outdir, "perfetto-trace.pfw")
    logger.info(f"Output will be saved to: {outfile}")
    if not os.path.exists(args.outdir):
        os.makedirs(args.outdir)

    # A trace set is a collection of event files
    traceset = TraceSet(events)
    if not traceset.files:
        logger.exit("No event files were found.")
    traceset.to_perfetto(outfile)
