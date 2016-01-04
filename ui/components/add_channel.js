import React from 'react';

import {
  Modal,
  Input,
  Button,
  ButtonGroup,
  ProgressBar
} from 'react-bootstrap';

import Icon from './icon';

export class AddChannelModal extends React.Component {

  constructor(props) {
    super(props);
    this.state = this.getDefaultState();
  }

  getDefaultState() {
    return {
      progress: 0,
      interval: null
    }
  }

  handleAdd(event){
    event.preventDefault();
    const node = this.refs.url.getInputDOMNode();
    this.props.onAdd(node.value);
    node.value = "";
  }

  componentWillReceiveProps(newProps) {
    if (newProps.pending && !this.props.pending) {
      this.setState({
        interval: window.setInterval(() => {
          this.setState({ progress: this.state.progress + 1 });
        }, 100)
      });
    } else if (!newProps.pending && this.props.pending) {
      window.clearInterval(this.state.interval);
      this.setState(this.getDefaultState());
    }
    return this.props !== newProps;
  }

  render() {
    const { show, onClose, container, pending } = this.props;

    const helpText = (
      <div>Enter the URL of the RSS feed you want to subscribe to, for example:
        <br /><em>http://joeroganexp.joerogan.libsynpro.com/rss</em>
      </div>
    );

    return (
      <Modal show={show}
             aria-labelledby="add-channel-modal-title"
             container={container}
             onHide={onClose}>
        <Modal.Header closeButton>
          <Modal.Title id="add-channel-modal-title">Add a new channel</Modal.Title>
        </Modal.Header>
        <Modal.Body>
            {pending ? (
            <div>
              <ProgressBar now={this.state.progress} />
            </div>
            ) : (
            <form className="form" onSubmit={this.handleAdd.bind(this)}>
              <Input required
                     type="text"
                     ref="url"
                     help={helpText} />
              <ButtonGroup>
              <Button bsStyle="primary" type="submit"><Icon icon="plus" /> Add channel</Button>
              <Button bsStyle="default" onClick={onClose}><Icon icon="remove" /> Cancel</Button>
            </ButtonGroup>
            </form>
            )}
        </Modal.Body>
      </Modal>
    );
  }

}

export default AddChannelModal;
